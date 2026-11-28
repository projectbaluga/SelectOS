#include <efi.h>
#include <efilib.h>
#include <efiprot.h>
#include <stdarg.h>
#define MAX_BOOT_ENTRIES 64
#define EFI_SIMPLE_FS_GUID { 0x964e5b22, 0x6459, 0x11d2, { 0x8e, 0x39, 0x00, 0xa0, 0xc9, 0x69, 0x72, 0x3b } }
static EFI_GUID SimpleFsGuid = EFI_SIMPLE_FS_GUID;
typedef struct { EFI_HANDLE Device; CHAR16 Path[128]; CHAR16 Label[64]; } BOOT_ENTRY;
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
            /* Retry with a volume-relative name for firmware implementations
             * whose root handles reject a leading backslash.
             */
            if (EFI_ERROR(Status)) {
                File = NULL;
                Status = uefi_call_wrapper(Root->Open, 5, Root, &File, L"selectos-debug.log",
                                           EFI_FILE_MODE_READ | EFI_FILE_MODE_WRITE | EFI_FILE_MODE_CREATE, 0);
            }
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
static VOID AppendChar(CHAR16 *Buffer, UINTN *Length, UINTN Capacity, CHAR16 Value) {
    if (*Length + 1 < Capacity) Buffer[(*Length)++] = Value;
}
static VOID AppendString(CHAR16 *Buffer, UINTN *Length, UINTN Capacity, const CHAR16 *Value) {
    UINTN i;
    if (Value == NULL) return;
    for (i = 0; Value[i] != 0 && *Length + 1 < Capacity; ++i)
        Buffer[(*Length)++] = Value[i];
}
static VOID AppendNumber(CHAR16 *Buffer, UINTN *Length, UINTN Capacity, UINTN Value) {
    CHAR16 Digits[24];
    UINTN Count = 0, i;
    do { Digits[Count++] = (CHAR16)(L'0' + (Value % 10)); Value /= 10; }
    while (Value != 0 && Count < sizeof(Digits) / sizeof(Digits[0]));
    for (i = Count; i > 0; --i) AppendChar(Buffer, Length, Capacity, Digits[i - 1]);
}
static VOID AppendHex(CHAR16 *Buffer, UINTN *Length, UINTN Capacity, UINTN Value) {
    static const CHAR16 Hex[] = L"0123456789ABCDEF";
    CHAR16 Digits[2 * sizeof(UINTN)];
    UINTN Count = 0, i;
    AppendString(Buffer, Length, Capacity, L"0x");
    do { Digits[Count++] = Hex[Value & 0x0f]; Value >>= 4; }
    while (Value != 0 && Count < sizeof(Digits) / sizeof(Digits[0]));
    for (i = Count; i > 0; --i) AppendChar(Buffer, Length, Capacity, Digits[i - 1]);
}
static UINTN ConsoleWriteHex(UINTN Value) {
    CHAR16 Buffer[2 * sizeof(UINTN) + 3];
    UINTN Length = 0;
    AppendHex(Buffer, &Length, sizeof(Buffer) / sizeof(Buffer[0]), Value);
    Buffer[Length] = 0;
    return ConsoleWriteString(Buffer);
}
/* Format into one UTF-16 buffer and call OutputString once. Some firmware
 * console implementations do not reliably handle a stream of one-character
 * OutputString calls, especially when output is redirected to a serial log.
 */
static UINTN LocalPrint(const CHAR16 *Format, ...) {
    CHAR16 Buffer[1024];
    UINTN Length = 0, i;
    va_list Args;
    va_start(Args, Format);
    for (i = 0; Format != NULL && Format[i] != 0 && Length + 1 < sizeof(Buffer) / sizeof(Buffer[0]); ++i) {
        if (Format[i] != L'%') {
            AppendChar(Buffer, &Length, sizeof(Buffer) / sizeof(Buffer[0]), Format[i]);
            continue;
        }
        ++i;
        if (Format[i] == 0) break;
        switch (Format[i]) {
            case L's':
                AppendString(Buffer, &Length, sizeof(Buffer) / sizeof(Buffer[0]), va_arg(Args, const CHAR16 *));
                break;
            case L'u':
                AppendNumber(Buffer, &Length, sizeof(Buffer) / sizeof(Buffer[0]), va_arg(Args, UINTN));
                break;
            case L'c':
                AppendChar(Buffer, &Length, sizeof(Buffer) / sizeof(Buffer[0]), (CHAR16)va_arg(Args, int));
                break;
            case L'r':
                AppendHex(Buffer, &Length, sizeof(Buffer) / sizeof(Buffer[0]), (UINTN)va_arg(Args, EFI_STATUS));
                break;
            case L'%':
                AppendChar(Buffer, &Length, sizeof(Buffer) / sizeof(Buffer[0]), L'%');
                break;
            default:
                AppendChar(Buffer, &Length, sizeof(Buffer) / sizeof(Buffer[0]), L'%');
                AppendChar(Buffer, &Length, sizeof(Buffer) / sizeof(Buffer[0]), Format[i]);
                break;
        }
    }
    va_end(Args);
    Buffer[Length] = 0;
    return ConsoleWriteString(Buffer);
}
#define KNOWN_LOADER_COUNT 10

static BOOLEAN CopyAsciiToChar16(const char *Source, CHAR16 *Destination, UINTN Capacity) {
    UINTN i = 0;
    if (Source == NULL || Destination == NULL || Capacity == 0) return FALSE;
    while (Source[i] != 0 && i + 1 < Capacity) {
        Destination[i] = (CHAR16)(UINT8)Source[i];
        ++i;
    }
    if (Source[i] != 0) return FALSE;
    Destination[i] = 0;
    return TRUE;
}
static UINTN LoaderPrefixCount(UINTN LoaderIndex) {
    if (LoaderIndex >= KNOWN_LOADER_COUNT) return 0;
    return LoaderIndex == 0 ? 4 : 3;
}
static BOOLEAN BuildLoaderPrefix(UINTN LoaderIndex, UINTN PrefixIndex, CHAR16 *Buffer, UINTN Capacity) {
    if (LoaderIndex >= KNOWN_LOADER_COUNT || PrefixIndex >= LoaderPrefixCount(LoaderIndex)) return FALSE;
    switch (LoaderIndex) {
        case 0:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\Microsoft", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\Microsoft\\Boot", Buffer, Capacity);
                case 3: return CopyAsciiToChar16("\\EFI\\Microsoft\\Boot\\bootmgfw.efi", Buffer, Capacity);
            }
            break;
        case 1:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\ubuntu", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\ubuntu\\shimx64.efi", Buffer, Capacity);
            }
            break;
        case 2:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\ubuntu", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\ubuntu\\grubx64.efi", Buffer, Capacity);
            }
            break;
        case 3:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\debian", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\debian\\shimx64.efi", Buffer, Capacity);
            }
            break;
        case 4:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\debian", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\debian\\grubx64.efi", Buffer, Capacity);
            }
            break;
        case 5:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\fedora", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\fedora\\shimx64.efi", Buffer, Capacity);
            }
            break;
        case 6:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\fedora", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\fedora\\grubx64.efi", Buffer, Capacity);
            }
            break;
        case 7:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\opensuse", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\opensuse\\shim.efi", Buffer, Capacity);
            }
            break;
        case 8:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\systemd", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\systemd\\systemd-bootx64.efi", Buffer, Capacity);
            }
            break;
        case 9:
            switch (PrefixIndex) {
                case 0: return CopyAsciiToChar16("\\EFI", Buffer, Capacity);
                case 1: return CopyAsciiToChar16("\\EFI\\refind", Buffer, Capacity);
                case 2: return CopyAsciiToChar16("\\EFI\\refind\\refind_x64.efi", Buffer, Capacity);
            }
            break;
    }
    return FALSE;
}
static BOOLEAN BuildLoaderPath(UINTN LoaderIndex, CHAR16 *Buffer, UINTN Capacity) {
    UINTN Count = LoaderPrefixCount(LoaderIndex);
    return Count != 0 && BuildLoaderPrefix(LoaderIndex, Count - 1, Buffer, Capacity);
}
static BOOLEAN BuildLoaderLabel(UINTN LoaderIndex, CHAR16 *Buffer, UINTN Capacity) {
    switch (LoaderIndex) {
        case 0: return CopyAsciiToChar16("Windows Boot Manager", Buffer, Capacity);
        case 1: return CopyAsciiToChar16("Ubuntu (shim)", Buffer, Capacity);
        case 2: return CopyAsciiToChar16("Ubuntu (GRUB)", Buffer, Capacity);
        case 3: return CopyAsciiToChar16("Debian (shim)", Buffer, Capacity);
        case 4: return CopyAsciiToChar16("Debian (GRUB)", Buffer, Capacity);
        case 5: return CopyAsciiToChar16("Fedora (shim)", Buffer, Capacity);
        case 6: return CopyAsciiToChar16("Fedora (GRUB)", Buffer, Capacity);
        case 7: return CopyAsciiToChar16("openSUSE (shim)", Buffer, Capacity);
        case 8: return CopyAsciiToChar16("systemd-boot", Buffer, Capacity);
        case 9: return CopyAsciiToChar16("rEFInd", Buffer, Capacity);
        default: return FALSE;
    }
}
static VOID PauseForKey(VOID) {
    EFI_INPUT_KEY key;
    while (uefi_call_wrapper(ST->ConIn->ReadKeyStroke, 2, ST->ConIn, &key) == EFI_NOT_READY)
        uefi_call_wrapper(BS->Stall, 1, 100000);
}
static EFI_FILE_HANDLE OpenPathFromRoot(EFI_FILE_HANDLE Root, UINTN LoaderIndex) {
    EFI_FILE_HANDLE Current = Root;
    BOOLEAN OwnsCurrent = FALSE;
    UINTN i, PrefixCount = LoaderPrefixCount(LoaderIndex);
    if (Root == NULL || PrefixCount == 0) return NULL;
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    if (LoaderIndex == 2) {
        CHAR16 Marker[] = L"SELECTOS_PATH_ENTRY\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    }
#endif
    /* Build writable UTF-16 path buffers from ASCII literals in this function. */
    for (i = 0; i < PrefixCount; ++i) {
        EFI_FILE_HANDLE Next = NULL;
        EFI_STATUS OpenStatus;
        CHAR16 FileName[128];
        if (!BuildLoaderPrefix(LoaderIndex, i, FileName, sizeof(FileName) / sizeof(FileName[0]))) {
            if (OwnsCurrent) uefi_call_wrapper(Current->Close, 1, Current);
            return NULL;
        }
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        if (LoaderIndex == 2) {
            if (i == 0) { CHAR16 M[] = L"SELECTOS_COMPONENT_EFI\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
            else if (i == 1) { CHAR16 M[] = L"SELECTOS_COMPONENT_UBUNTU\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
            else if (i == 2) { CHAR16 M[] = L"SELECTOS_COMPONENT_GRUB\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
        }
#endif
        OpenStatus = uefi_call_wrapper(Root->Open, 5, Root, &Next,
                                       FileName, (UINT64)EFI_FILE_MODE_READ, (UINT64)0);
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        if (LoaderIndex == 2) {
            if (EFI_ERROR(OpenStatus) || Next == NULL) {
                if (OpenStatus == EFI_NOT_FOUND) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_NOT_FOUND\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else if (OpenStatus == EFI_ACCESS_DENIED) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_ACCESS_DENIED\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else if (OpenStatus == EFI_INVALID_PARAMETER) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_INVALID_PARAMETER\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else if (OpenStatus == EFI_UNSUPPORTED) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_UNSUPPORTED\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else if (OpenStatus == EFI_DEVICE_ERROR) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_DEVICE_ERROR\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else if (OpenStatus == EFI_NO_MEDIA) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_NO_MEDIA\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else if (OpenStatus == EFI_MEDIA_CHANGED) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_MEDIA_CHANGED\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else if (OpenStatus == EFI_TIMEOUT) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_TIMEOUT\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else if (OpenStatus == EFI_COMPROMISED_DATA) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_COMPROMISED_DATA\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else if (OpenStatus == EFI_SUCCESS && Next == NULL) { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_SUCCESS_NULL_HANDLE\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
                else { CHAR16 M[] = L"SELECTOS_COMPONENT_STATUS_OTHER\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M); }
            } else {
                CHAR16 M[] = L"SELECTOS_PATH_COMPONENT_OPENED\r\n";
                uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, M);
            }
        }
#endif
        if (EFI_ERROR(OpenStatus) || Next == NULL) {
            if (OwnsCurrent) uefi_call_wrapper(Current->Close, 1, Current);
            return NULL;
        }
        if (OwnsCurrent) uefi_call_wrapper(Current->Close, 1, Current);
        Current = Next;
        OwnsCurrent = TRUE;
    }
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    if (LoaderIndex == 2) {
        CHAR16 Marker[] = L"SELECTOS_PATH_FINAL_OPENED\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    }
#endif
    return Current;
}
static BOOLEAN FileExists(EFI_FILE_HANDLE Root, UINTN LoaderIndex) {
    EFI_FILE_HANDLE File = OpenPathFromRoot(Root, LoaderIndex);
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    if (LoaderIndex == 2) {
        CHAR16 Marker[] = L"SELECTOS_TEST_CHILD_PATH_OPENED\r\n";
        CHAR16 Missing[] = L"SELECTOS_TEST_CHILD_PATH_NOT_FOUND\r\n";
        if (File != NULL) uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
        else uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Missing);
    }
#endif
    if (File == NULL) return FALSE;
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
        CHAR16 Marker[] = L"SELECTOS_SCAN_HANDLE_PROTOCOL_FAILED\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
        return;
    }
    Status = uefi_call_wrapper(Fs->OpenVolume, 2, Fs, &Root);
    if (EFI_ERROR(Status) || Root == NULL) {
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        CHAR16 Marker[] = L"SELECTOS_SCAN_OPEN_VOLUME_FAILED\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
        return;
    }
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    {
        CHAR16 Marker[] = L"SELECTOS_SCAN_VOLUME_OPENED\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    }
#endif
    for (j = 0; j < KNOWN_LOADER_COUNT; ++j) {
        if (EntryCount >= MAX_BOOT_ENTRIES) break;
        if (FileExists(Root, j)) {
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
            CHAR16 Marker[] = L"SELECTOS_SCAN_ALLOWLISTED_LOADER_FOUND\r\n";
            uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
            Entries[EntryCount].Device = Device;
            if (!BuildLoaderPath(j, Entries[EntryCount].Path, sizeof(Entries[EntryCount].Path) / sizeof(Entries[EntryCount].Path[0])) ||
                !BuildLoaderLabel(j, Entries[EntryCount].Label, sizeof(Entries[EntryCount].Label) / sizeof(Entries[EntryCount].Label[0]))) continue;
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
        CHAR16 Marker[] = L"SELECTOS_SCAN_LOADED_IMAGE_VOLUME\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
        ScanVolume(LoadedImage->DeviceHandle);
    } else {
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        CHAR16 Marker[] = L"SELECTOS_SCAN_NO_LOADED_IMAGE_VOLUME\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
#endif
    }

    Status = uefi_call_wrapper(BS->LocateHandleBuffer, 5, ByProtocol, &SimpleFsGuid, NULL, &HandleCount, &Handles);
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
    if (EFI_ERROR(Status)) {
        CHAR16 Marker[] = L"SELECTOS_SCAN_LOCATE_FAILED\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    } else if (HandleCount == 0 || Handles == NULL) {
        CHAR16 Marker[] = L"SELECTOS_SCAN_NO_FS_HANDLES\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    } else {
        CHAR16 Marker[] = L"SELECTOS_SCAN_FS_HANDLES_FOUND\r\n";
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
        CHAR16 Marker[] = L"SELECTOS_SCAN_ZERO_ENTRIES\r\n";
        uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker);
    } else {
        CHAR16 Marker[] = L"SELECTOS_SCAN_ENTRIES_FOUND\r\n";
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
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        { CHAR16 Marker[] = L"SELECTOS_LOOP_ENTERED\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker); }
#endif
        UINTN i, Selection;
        EFI_STATUS Status;
#ifndef SELECTOS_QEMU_TEST_DIAGNOSTICS
        uefi_call_wrapper(ST->ConOut->ClearScreen, 1, ST->ConOut);
#endif
        LocalPrint(L"SelectOS UEFI Boot Manager — experimental\r\n");
        LocalPrint(L"===========================================\r\n\r\n");
        LocalPrint(L"Scanning UEFI-readable filesystem volumes...\r\n");
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        { CHAR16 Marker[] = L"SELECTOS_SCAN_CALL_BEGIN\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker); }
#endif
        ScanFilesystems(ImageHandle);
#ifdef SELECTOS_QEMU_TEST_DIAGNOSTICS
        { CHAR16 Marker[] = L"SELECTOS_SCAN_CALL_END\r\n"; uefi_call_wrapper(ST->ConOut->OutputString, 2, ST->ConOut, Marker); }
#endif
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
