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
static EFI_STATUS LaunchEntry(EFI_HANDLE ImageHandle, const BOOT_ENTRY *Entry) {
    EFI_DEVICE_PATH_PROTOCOL *DevicePath;
    EFI_HANDLE ChildImage = NULL;
    EFI_STATUS Status;
    UINTN ExitDataSize = 0;
    CHAR16 *ExitData = NULL;
    DevicePath = FileDevicePath(Entry->Device, (CHAR16 *)Entry->Path);
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
