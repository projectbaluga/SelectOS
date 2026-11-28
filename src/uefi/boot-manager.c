#include <efi.h>
#include <efilib.h>
#include <efiprot.h>
#define MAX_BOOT_ENTRIES 64
#define EFI_SIMPLE_FS_GUID { 0x964e5b22, 0x6459, 0x11d2, { 0x8e, 0x39, 0x00, 0xa0, 0xc9, 0x69, 0x72, 0x3b } }
static EFI_GUID SimpleFsGuid = EFI_SIMPLE_FS_GUID;
typedef struct { EFI_HANDLE Device; const CHAR16 *Path; const CHAR16 *Label; } BOOT_ENTRY;
static BOOT_ENTRY Entries[MAX_BOOT_ENTRIES];
static UINTN EntryCount;
static const struct { const CHAR16 *Path; const CHAR16 *Label; } KnownLoaders[] = {
    { L"\\EFI\\Microsoft\\Boot\\bootmgfw.efi", L"Windows Boot Manager" },
    { L"\\EFI\\ubuntu\\shimx64.efi", L"Ubuntu (shim)" },
    { L"\\EFI\\ubuntu\\grubx64.efi", L"Ubuntu (GRUB)" },
    { L"\\EFI\\debian\\shimx64.efi", L"Debian (shim)" },
    { L"\\EFI\\debian\\grubx64.efi", L"Debian (GRUB)" },
    { L"\\EFI\\fedora\\shimx64.efi", L"Fedora (shim)" },
    { L"\\EFI\\fedora\\grubx64.efi", L"Fedora (GRUB)" },
    { L"\\EFI\\opensuse\\shim.efi", L"openSUSE (shim)" },
    { L"\\EFI\\systemd\\systemd-bootx64.efi", L"systemd-boot" },
    { L"\\EFI\\refind\\refind_x64.efi", L"rEFInd" }
};

static VOID PauseForKey(VOID) {
    EFI_INPUT_KEY key;
    while (uefi_call_wrapper(ST->ConIn->ReadKeyStroke, 2, ST->ConIn, &key) == EFI_NOT_READY)
        uefi_call_wrapper(BS->Stall, 1, 100000);
}
static BOOLEAN FileExists(EFI_FILE_HANDLE Root, const CHAR16 *Path) {
    EFI_FILE_HANDLE File = NULL;
    EFI_STATUS Status = uefi_call_wrapper(Root->Open, 5, Root, &File, (CHAR16 *)Path, EFI_FILE_MODE_READ, 0);
    if (EFI_ERROR(Status)) return FALSE;
    uefi_call_wrapper(File->Close, 1, File);
    return TRUE;
}
static VOID ScanFilesystems(VOID) {
    EFI_HANDLE *Handles = NULL;
    UINTN HandleCount = 0, i, j;
    EFI_STATUS Status;
    EntryCount = 0;
    Status = uefi_call_wrapper(BS->LocateHandleBuffer, 5, ByProtocol, &SimpleFsGuid, NULL, &HandleCount, &Handles);
    if (EFI_ERROR(Status) || Handles == NULL) return;
    for (i = 0; i < HandleCount && EntryCount < MAX_BOOT_ENTRIES; ++i) {
        EFI_SIMPLE_FILE_SYSTEM_PROTOCOL *Fs = NULL;
        EFI_FILE_HANDLE Root = NULL;
        Status = uefi_call_wrapper(BS->HandleProtocol, 3, Handles[i], &SimpleFsGuid, (VOID **)&Fs);
        if (EFI_ERROR(Status) || Fs == NULL) continue;
        Status = uefi_call_wrapper(Fs->OpenVolume, 2, Fs, &Root);
        if (EFI_ERROR(Status) || Root == NULL) continue;
        for (j = 0; j < sizeof(KnownLoaders) / sizeof(KnownLoaders[0]); ++j) {
            if (EntryCount >= MAX_BOOT_ENTRIES) break;
            if (FileExists(Root, KnownLoaders[j].Path)) {
                Entries[EntryCount].Device = Handles[i];
                Entries[EntryCount].Path = KnownLoaders[j].Path;
                Entries[EntryCount].Label = KnownLoaders[j].Label;
                ++EntryCount;
            }
        }
        uefi_call_wrapper(Root->Close, 1, Root);
    }
    uefi_call_wrapper(BS->FreePool, 1, Handles);
}
static UINTN ReadSelection(VOID) {
    EFI_INPUT_KEY Key;
    UINTN Value = 0;
    BOOLEAN HasDigits = FALSE;
    Print(L"\r\nEnter entry number, or q to return to GRUB: ");
    for (;;) {
        EFI_STATUS Status;
        do {
            Status = uefi_call_wrapper(ST->ConIn->ReadKeyStroke, 2, ST->ConIn, &Key);
            if (Status == EFI_NOT_READY) uefi_call_wrapper(BS->Stall, 1, 100000);
        } while (Status == EFI_NOT_READY);
        if (EFI_ERROR(Status)) continue;
        if (Key.UnicodeChar == L'q' || Key.UnicodeChar == L'Q') return 0;
        if (Key.UnicodeChar == L'\r') return HasDigits ? Value : 0;
        if (Key.UnicodeChar >= L'0' && Key.UnicodeChar <= L'9') {
            UINTN Digit = (UINTN)(Key.UnicodeChar - L'0');
            if (Value > 999) continue;
            Value = Value * 10 + Digit;
            HasDigits = TRUE;
            Print(L"%c", Key.UnicodeChar);
        } else if (Key.UnicodeChar == L'\b' && HasDigits) {
            Value /= 10;
            Print(L"\b \b");
        }
    }
}
#ifndef MEDIA_DEVICE_PATH
#define MEDIA_DEVICE_PATH 0x04
#endif
#ifndef MEDIA_FILEPATH_DP
#define MEDIA_FILEPATH_DP 0x04
#endif
#ifndef END_DEVICE_PATH_TYPE
#define END_DEVICE_PATH_TYPE 0x7f
#endif
#ifndef END_ENTIRE_DEVICE_PATH_SUBTYPE
#define END_ENTIRE_DEVICE_PATH_SUBTYPE 0xff
#endif

/* Build a file device path locally instead of using GNU-EFI's FileDevicePath.
 * Some distro-built GNU-EFI libraries use newer SIMD instructions than the
 * baseline x86-64 CPU exposed by common firmware and emulators.
 */
static EFI_DEVICE_PATH_PROTOCOL *BuildFileDevicePath(EFI_HANDLE Device, const CHAR16 *Path) {
    EFI_DEVICE_PATH_PROTOCOL *Base = NULL, *Node, *Result;
    EFI_STATUS Status;
    UINTN BaseSize = 0, NodeLength, PathChars = 0, FileNodeSize, BasePrefix, TotalSize, i;
    Status = uefi_call_wrapper(BS->HandleProtocol, 3, Device, &DevicePathProtocol, (VOID **)&Base);
    if (EFI_ERROR(Status) || Base == NULL) return NULL;

    Node = Base;
    for (;;) {
        NodeLength = (UINTN)Node->Length[0] | ((UINTN)Node->Length[1] << 8);
        if (NodeLength < sizeof(EFI_DEVICE_PATH_PROTOCOL)) return NULL;
        if (Node->Type == END_DEVICE_PATH_TYPE && Node->SubType == END_ENTIRE_DEVICE_PATH_SUBTYPE) {
            if (NodeLength != sizeof(EFI_DEVICE_PATH_PROTOCOL)) return NULL;
            BaseSize += NodeLength;
            break;
        }
        BaseSize += NodeLength;
        if (BaseSize > 65536) return NULL;
        Node = (EFI_DEVICE_PATH_PROTOCOL *)((UINT8 *)Node + NodeLength);
    }

    while (Path[PathChars] != 0) {
        ++PathChars;
        if (PathChars > 4096) return NULL;
    }
    FileNodeSize = sizeof(EFI_DEVICE_PATH_PROTOCOL) + (PathChars + 1) * sizeof(CHAR16);
    if (FileNodeSize > 0xffff) return NULL;
    BasePrefix = BaseSize - sizeof(EFI_DEVICE_PATH_PROTOCOL);
    TotalSize = BasePrefix + FileNodeSize + sizeof(EFI_DEVICE_PATH_PROTOCOL);
    Status = uefi_call_wrapper(BS->AllocatePool, 3, EfiBootServicesData, TotalSize, (VOID **)&Result);
    if (EFI_ERROR(Status) || Result == NULL) return NULL;

    for (i = 0; i < BasePrefix; ++i) ((UINT8 *)Result)[i] = ((UINT8 *)Base)[i];
    Node = (EFI_DEVICE_PATH_PROTOCOL *)((UINT8 *)Result + BasePrefix);
    Node->Type = MEDIA_DEVICE_PATH;
    Node->SubType = MEDIA_FILEPATH_DP;
    Node->Length[0] = (UINT8)(FileNodeSize & 0xff);
    Node->Length[1] = (UINT8)((FileNodeSize >> 8) & 0xff);
    {
        CHAR16 *Destination = (CHAR16 *)((UINT8 *)Node + sizeof(EFI_DEVICE_PATH_PROTOCOL));
        for (i = 0; i <= PathChars; ++i) Destination[i] = Path[i];
    }
    Node = (EFI_DEVICE_PATH_PROTOCOL *)((UINT8 *)Node + FileNodeSize);
    Node->Type = END_DEVICE_PATH_TYPE;
    Node->SubType = END_ENTIRE_DEVICE_PATH_SUBTYPE;
    Node->Length[0] = (UINT8)sizeof(EFI_DEVICE_PATH_PROTOCOL);
    Node->Length[1] = 0;
    return Result;
}

static EFI_STATUS LaunchEntry(EFI_HANDLE ImageHandle, const BOOT_ENTRY *Entry) {
    EFI_DEVICE_PATH_PROTOCOL *DevicePath;
    EFI_HANDLE ChildImage = NULL;
    EFI_STATUS Status;
    UINTN ExitDataSize = 0;
    CHAR16 *ExitData = NULL;
    DevicePath = BuildFileDevicePath(Entry->Device, Entry->Path);
    if (DevicePath == NULL) return EFI_OUT_OF_RESOURCES;
    Status = uefi_call_wrapper(BS->LoadImage, 6, FALSE, ImageHandle, DevicePath, NULL, 0, &ChildImage);
    uefi_call_wrapper(BS->FreePool, 1, DevicePath);
    if (EFI_ERROR(Status)) return Status;
    Print(L"\r\nStarting %s...\r\n", Entry->Label);
    Status = uefi_call_wrapper(BS->StartImage, 3, ChildImage, &ExitDataSize, &ExitData);
    if (ExitData != NULL) {
        Print(L"Loader returned: %s\r\n", ExitData);
        uefi_call_wrapper(BS->FreePool, 1, ExitData);
    }
    uefi_call_wrapper(BS->UnloadImage, 1, ChildImage);
    return Status;
}
EFI_STATUS EFIAPI efi_main(EFI_HANDLE ImageHandle, EFI_SYSTEM_TABLE *SystemTable) {
    InitializeLib(ImageHandle, SystemTable);
    for (;;) {
        UINTN i, Selection;
        EFI_STATUS Status;
        uefi_call_wrapper(ST->ConOut->ClearScreen, 1, ST->ConOut);
        Print(L"SelectOS UEFI Boot Manager — experimental\r\n");
        Print(L"===========================================\r\n\r\n");
        Print(L"Scanning UEFI-readable filesystem volumes...\r\n");
        ScanFilesystems();
        if (EntryCount == 0) {
            Print(L"\r\nNo supported EFI loader was found.\r\n");
            Print(L"No firmware boot variable or disk content was changed.\r\n");
            Print(L"\r\nPress any key to return to GRUB.\r\n");
            PauseForKey();
            return EFI_NOT_FOUND;
        }
        Print(L"\r\nAvailable operating-system loaders:\r\n");
        for (i = 0; i < EntryCount; ++i)
            Print(L"  %u) %s — %s\r\n", (UINTN)(i + 1), Entries[i].Label, Entries[i].Path);
        Print(L"\r\nFirmware Secure Boot policy remains in force.\r\n");
        Print(L"SelectOS does not bypass image signature checks.\r\n");
        Selection = ReadSelection();
        if (Selection == 0) return EFI_SUCCESS;
        if (Selection > EntryCount) {
            Print(L"\r\nInvalid selection. Press any key to retry.\r\n");
            PauseForKey();
            continue;
        }
        Status = LaunchEntry(ImageHandle, &Entries[Selection - 1]);
        if (Status == EFI_SECURITY_VIOLATION) {
            Print(L"\r\nFirmware rejected this loader under its security policy.\r\n");
            Print(L"Use a trusted signed loader; Secure Boot was not changed.\r\n");
        } else if (EFI_ERROR(Status)) {
            Print(L"\r\nCould not start the selected loader: %r\r\n", Status);
        } else {
            Print(L"\r\nThe selected loader returned control to SelectOS.\r\n");
        }
        Print(L"Press any key to return to the menu.\r\n");
        PauseForKey();
    }
}
