#include <efi.h>
#include <efilib.h>
#include <efiprot.h>
#include <stdarg.h>
#define MAX_BOOT_ENTRIES 64
#define EFI_SIMPLE_FS_GUID { 0x964e5b22, 0x6459, 0x11d2, { 0x8e, 0x39, 0x00, 0xa0, 0xc9, 0x69, 0x72, 0x3b } }
static EFI_GUID SimpleFsGuid = EFI_SIMPLE_FS_GUID;
typedef struct { EFI_HANDLE Device; const CHAR16 *Path; const CHAR16 *Label; } BOOT_ENTRY;
static BOOT_ENTRY Entries[MAX_BOOT_ENTRIES];
static UINTN EntryCount;
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
static VOID WriteTestDiagnostics(VOID) {
    EFI_HANDLE *Handles = NULL;
    UINTN HandleCount = 0, i;
    EFI_STATUS Status;
    CHAR8 Buffer[128];
    UINTN Length = 0;
    const CHAR8 Prefix[] = "SELECTOS_MANAGER_STARTED\n";
    const CHAR8 Scan[] = "SCAN_HANDLES=";
    const CHAR8 EntriesText[] = "\nENTRIES=";
    Status = uefi_call_wrapper(BS->LocateHandleBuffer, 5, ByProtocol, &SimpleFsGuid,
                               NULL, &HandleCount, &Handles);
    for (i = 0; i + 1 < sizeof(Prefix); ++i) Buffer[Length++] = Prefix[i];
    for (i = 0; i + 1 < sizeof(Scan); ++i) Buffer[Length++] = Scan[i];
    {
        CHAR8 Digits[24]; UINTN Count = 0, Value = HandleCount, j;
        do { Digits[Count++] = (CHAR8)('0' + (Value % 10)); Value /= 10; } while (Value && Count < sizeof(Digits));
        for (j = Count; j > 0; --j) Buffer[Length++] = Digits[j - 1];
    }
    for (i = 0; i + 1 < sizeof(EntriesText); ++i) Buffer[Length++] = EntriesText[i];
    {
        CHAR8 Digits[24]; UINTN Count = 0, Value = EntryCount, j;
        do { Digits[Count++] = (CHAR8)('0' + (Value % 10)); Value /= 10; } while (Value && Count < sizeof(Digits));
        for (j = Count; j > 0; --j) Buffer[Length++] = Digits[j - 1];
    }
    Buffer[Length++] = '\n';
    if (!EFI_ERROR(Status) && Handles != NULL) {
        for (i = 0; i < HandleCount; ++i) {
            EFI_SIMPLE_FILE_SYSTEM_PROTOCOL *Fs = NULL;
            EFI_FILE_HANDLE Root = NULL, File = NULL;
            UINTN WriteLength = Length;
            Status = uefi_call_wrapper(BS->HandleProtocol, 3, Handles[i], &SimpleFsGuid,
                                       (VOID **)&Fs);
            if (EFI_ERROR(Status) || Fs == NULL) continue;
            Status = uefi_call_wrapper(Fs->OpenVolume, 2, Fs, &Root);
            if (EFI_ERROR(Status) || Root == NULL) continue;
            Status = uefi_call_wrapper(Root->Open, 5, Root, &File, L"\\selectos-debug.log",
                                       EFI_FILE_MODE_READ | EFI_FILE_MODE_WRITE | EFI_FILE_MODE_CREATE, 0);
            if (!EFI_ERROR(Status) && File != NULL) {
                uefi_call_wrapper(File->SetPosition, 2, File, 0);
                uefi_call_wrapper(File->Write, 3, File, &WriteLength, Buffer);
                uefi_call_wrapper(File->Flush, 1, File);
                uefi_call_wrapper(File->Close, 1, File);
            }
            uefi_call_wrapper(Root->Close, 1, Root);
        }
        uefi_call_wrapper(BS->FreePool, 1, Handles);
    }
}
#endif
static UINTN ConsoleWriteString(const CHAR16 *Text) {
    UINTN Count = 0;
    if (Text == NULL || ST == NULL || ST->ConOut == NULL) return 0;
    while (Text[Count] != 0) ++Count;
    uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, (CHAR16 *)Text);
    return Count;
}
static VOID ConsoleWriteChar(CHAR16 Character) {
    CHAR16 Text[2];
    Text[0] = Character;
    Text[1] = 0;
    if (ST != NULL && ST->ConOut != NULL)
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Text);
}
static UINTN ConsoleWriteNumber(UINTN Value) {
    CHAR16 Digits[24];
    UINTN Count = 0, i;
    if (Value == 0) {
        ConsoleWriteChar(L'0');
        return 1;
    }
    while (Value != 0 && Count < sizeof(Digits) / sizeof(Digits[0])) {
        Digits[Count++] = (CHAR16)(L'0' + (Value % 10));
        Value /= 10;
    }
    for (i = Count; i > 0; --i) ConsoleWriteChar(Digits[i - 1]);
    return Count;
}
static UINTN ConsoleWriteHex(UINTN Value) {
    static const CHAR16 Hex[] = L"0123456789ABCDEF";
    CHAR16 Digits[2 * sizeof(UINTN)];
    UINTN Count = 0, i;
    ConsoleWriteString(L"0x");
    do {
        Digits[Count++] = Hex[Value & 0x0f];
        Value >>= 4;
    } while (Value != 0 && Count < sizeof(Digits) / sizeof(Digits[0]));
    for (i = Count; i > 0; --i) ConsoleWriteChar(Digits[i - 1]);
    return Count + 2;
}
/* Minimal local formatter: avoid GNU-EFI Print/CatPrint library code, which
 * may contain instructions newer than the baseline x86-64 ISA.
 */
static UINTN LocalPrint(const CHAR16 *Format, ...) {
    va_list Args;
    UINTN Count = 0, i;
    va_start(Args, Format);
    for (i = 0; Format != NULL && Format[i] != 0; ++i) {
        if (Format[i] != L'%') {
            ConsoleWriteChar(Format[i]);
            ++Count;
            continue;
        }
        ++i;
        if (Format[i] == 0) break;
        switch (Format[i]) {
            case L's': {
                const CHAR16 *Text = va_arg(Args, const CHAR16 *);
                Count += ConsoleWriteString(Text);
                break;
            }
            case L'u':
                Count += ConsoleWriteNumber(va_arg(Args, UINTN));
                break;
            case L'c':
                ConsoleWriteChar((CHAR16)va_arg(Args, int));
                ++Count;
                break;
            case L'r':
                Count += ConsoleWriteHex((UINTN)va_arg(Args, EFI_STATUS));
                break;
            case L'%':
                ConsoleWriteChar(L'%');
                ++Count;
                break;
            default:
                ConsoleWriteChar(L'%');
                ConsoleWriteChar(Format[i]);
                Count += 2;
                break;
        }
    }
    va_end(Args);
    return Count;
}
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
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    if (Path == KnownLoaders[2].Path) {
        CHAR16 Marker[] = L"SELECTOS_TEST_CHILD_PATH_OPENED\\r\\n";
        CHAR16 Missing[] = L"SELECTOS_TEST_CHILD_PATH_NOT_FOUND\\r\\n";
        CHAR16 Failed[] = L"SELECTOS_TEST_CHILD_PATH_OPEN_FAILED\\r\\n";
        if (!EFI_ERROR(Status)) uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
        else if (Status == EFI_NOT_FOUND) uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Missing);
        else uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Failed);
    }
#endif
    if (EFI_ERROR(Status)) return FALSE;
    uefi_call_wrapper(File->Close, 1, File);
    return TRUE;
}
static VOID ScanVolume(EFI_HANDLE Device) {
    EFI_SIMPLE_FILE_SYSTEM_PROTOCOL *Fs = NULL;
    EFI_FILE_HANDLE Root = NULL;
    EFI_STATUS Status;
    UINTN j;
    if (Device == NULL || EntryCount >= MAX_BOOT_ENTRIES) return;
    Status = uefi_call_wrapper(BS->HandleProtocol, 3, Device, &SimpleFsGuid, (VOID **)&Fs);
    if (EFI_ERROR(Status) || Fs == NULL) {
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        CHAR16 Marker[] = L"SELECTOS_SCAN_HANDLE_PROTOCOL_FAILED\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
        return;
    }
    Status = uefi_call_wrapper(Fs->OpenVolume, 2, Fs, &Root);
    if (EFI_ERROR(Status) || Root == NULL) {
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        CHAR16 Marker[] = L"SELECTOS_SCAN_OPEN_VOLUME_FAILED\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
        return;
    }
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    {
        CHAR16 Marker[] = L"SELECTOS_SCAN_VOLUME_OPENED\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    }
#endif
    for (j = 0; j < sizeof(KnownLoaders) / sizeof(KnownLoaders[0]); ++j) {
        if (EntryCount >= MAX_BOOT_ENTRIES) break;
        if (FileExists(Root, KnownLoaders[j].Path)) {
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
            CHAR16 Marker[] = L"SELECTOS_SCAN_ALLOWLISTED_LOADER_FOUND\\r\\n";
            uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
            Entries[EntryCount].Device = Device;
            Entries[EntryCount].Path = KnownLoaders[j].Path;
            Entries[EntryCount].Label = KnownLoaders[j].Label;
            ++EntryCount;
        }
    }
    uefi_call_wrapper(Root->Close, 1, Root);
}
static VOID ScanFilesystems(EFI_HANDLE ImageHandle) {
    EFI_HANDLE *Handles = NULL;
    EFI_LOADED_IMAGE *LoadedImage = NULL;
    UINTN HandleCount = 0, i;
    EFI_STATUS Status;
    EntryCount = 0;

    /* Prefer the volume from which this manager was loaded. This is the most
     * relevant filesystem for a chainloaded application and avoids depending
     * solely on global filesystem enumeration order.
     */
    Status = uefi_call_wrapper(BS->HandleProtocol, 3, ImageHandle,
                               &LoadedImageProtocol, (VOID **)&LoadedImage);
    if (!EFI_ERROR(Status) && LoadedImage != NULL && LoadedImage->DeviceHandle != NULL) {
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        CHAR16 Marker[] = L"SELECTOS_SCAN_LOADED_IMAGE_VOLUME\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
        ScanVolume(LoadedImage->DeviceHandle);
    } else {
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        CHAR16 Marker[] = L"SELECTOS_SCAN_NO_LOADED_IMAGE_VOLUME\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
    }

    Status = uefi_call_wrapper(BS->LocateHandleBuffer, 5, ByProtocol, &SimpleFsGuid, NULL, &HandleCount, &Handles);
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    if (EFI_ERROR(Status)) {
        CHAR16 Marker[] = L"SELECTOS_SCAN_LOCATE_FAILED\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    } else if (HandleCount == 0 || Handles == NULL) {
        CHAR16 Marker[] = L"SELECTOS_SCAN_NO_FS_HANDLES\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    } else {
        CHAR16 Marker[] = L"SELECTOS_SCAN_FS_HANDLES_FOUND\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    }
#endif
    if (!EFI_ERROR(Status) && Handles != NULL) {
        for (i = 0; i < HandleCount && EntryCount < MAX_BOOT_ENTRIES; ++i) {
            if (LoadedImage != NULL && Handles[i] == LoadedImage->DeviceHandle) continue;
            ScanVolume(Handles[i]);
        }
        uefi_call_wrapper(BS->FreePool, 1, Handles);
    }
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    if (EntryCount == 0) {
        CHAR16 Marker[] = L"SELECTOS_SCAN_ZERO_ENTRIES\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    } else {
        CHAR16 Marker[] = L"SELECTOS_SCAN_ENTRIES_FOUND\\r\\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    }
#endif
}
static UINTN ReadSelection(VOID) {
    EFI_INPUT_KEY Key;
    UINTN Value = 0;
    BOOLEAN HasDigits = FALSE;
    LocalPrint(L"\r\nEnter entry number, or q to return to GRUB: ");
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
            LocalPrint(L"%c", Key.UnicodeChar);
        } else if (Key.UnicodeChar == L'\b' && HasDigits) {
            Value /= 10;
            LocalPrint(L"\b \b");
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
    LocalPrint(L"\r\nStarting %s...\r\n", Entry->Label);
    Status = uefi_call_wrapper(BS->StartImage, 3, ChildImage, &ExitDataSize, &ExitData);
    if (ExitData != NULL) {
        LocalPrint(L"Loader returned: %s\r\n", ExitData);
        uefi_call_wrapper(BS->FreePool, 1, ExitData);
    }
    uefi_call_wrapper(BS->UnloadImage, 1, ChildImage);
    return Status;
}
EFI_STATUS EFIAPI efi_main(EFI_HANDLE ImageHandle, EFI_SYSTEM_TABLE *SystemTable) {
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    if (SystemTable != NULL && SystemTable->ConOut != NULL) {
        CHAR16 Marker[] = L"SELECTOS_MANAGER_ENTRYPOINT_REACHED\r\n";
        uefi_call_wrapper(SystemTable->ConOut->OutputString, 2, SystemTable->ConOut, Marker);
    }
#endif
    InitializeLib(ImageHandle, SystemTable);
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    {
        CHAR16 Marker[] = L"SELECTOS_MANAGER_INITIALIZED\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    }
    WriteTestDiagnostics();
    {
        CHAR16 Marker[] = L"SELECTOS_DIAGNOSTICS_RETURNED\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    }
#endif
    for (;;) {
        UINTN i, Selection;
        EFI_STATUS Status;
#ifndef SELECTOS_QEMU_TEST_DIAGNOSTICS
        uefi_call_wrapper(ST->ConOut->ClearScreen, 1, ST->ConOut);
#endif
        LocalPrint(L"SelectOS UEFI Boot Manager — experimental\r\n");
        LocalPrint(L"===========================================\r\n\r\n");
        LocalPrint(L"Scanning UEFI-readable filesystem volumes...\r\n");
        ScanFilesystems(ImageHandle);
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        {
            CHAR16 Marker[] = L"SELECTOS_SCAN_RETURNED\r\n";
            uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
        }
        WriteTestDiagnostics();
        {
            CHAR16 Marker[] = L"SELECTOS_POSTSCAN_DIAGNOSTICS_RETURNED\r\n";
            uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
        }
#endif
        if (EntryCount == 0) {
            LocalPrint(L"\r\nNo supported EFI loader was found.\r\n");
            LocalPrint(L"No firmware boot variable or disk content was changed.\r\n");
            LocalPrint(L"\r\nPress any key to return to GRUB.\r\n");
            PauseForKey();
            return EFI_NOT_FOUND;
        }
        LocalPrint(L"\r\nAvailable operating-system loaders:\r\n");
        for (i = 0; i < EntryCount; ++i)
            LocalPrint(L"  %u) %s — %s\r\n", (UINTN)(i + 1), Entries[i].Label, Entries[i].Path);
        LocalPrint(L"\r\nFirmware Secure Boot policy remains in force.\r\n");
        LocalPrint(L"SelectOS does not bypass image signature checks.\r\n");
        Selection = ReadSelection();
        if (Selection == 0) return EFI_SUCCESS;
        if (Selection > EntryCount) {
            LocalPrint(L"\r\nInvalid selection. Press any key to retry.\r\n");
            PauseForKey();
            continue;
        }
        Status = LaunchEntry(ImageHandle, &Entries[Selection - 1]);
        if (Status == EFI_SECURITY_VIOLATION) {
            LocalPrint(L"\r\nFirmware rejected this loader under its security policy.\r\n");
            LocalPrint(L"Use a trusted signed loader; Secure Boot was not changed.\r\n");
        } else if (EFI_ERROR(Status)) {
            LocalPrint(L"\r\nCould not start the selected loader: %r\r\n", Status);
        } else {
            LocalPrint(L"\r\nThe selected loader returned control to SelectOS.\r\n");
        }
        LocalPrint(L"Press any key to return to the menu.\r\n");
        PauseForKey();
    }
}
