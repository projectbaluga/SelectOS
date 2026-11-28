#include <efi.h>
#include <efilib.h>
#include <efiprot.h>

EFI_STATUS EFIAPI efi_main(EFI_HANDLE ImageHandle, EFI_SYSTEM_TABLE *SystemTable) {
    EFI_LOADED_IMAGE *LoadedImage = NULL;
    EFI_SIMPLE_FILE_SYSTEM_PROTOCOL *FileSystem = NULL;
    EFI_FILE_HANDLE Root = NULL, Marker = NULL;
    EFI_STATUS Status;
    CHAR16 Path[] = L"\\handoff.ok";
    CHAR8 Payload[] = "SELECTOS_QEMU_OVMF_HANDOFF_OK\n";
    UINTN PayloadSize = sizeof(Payload) - 1;

    InitializeLib(ImageHandle, SystemTable);
    Status = uefi_call_wrapper(BS->HandleProtocol, 3, ImageHandle,
                               &LoadedImageProtocol, (VOID **)&LoadedImage);
    if (EFI_ERROR(Status) || LoadedImage == NULL) return EFI_LOAD_ERROR;
    Status = uefi_call_wrapper(BS->HandleProtocol, 3, LoadedImage->DeviceHandle,
                               &FileSystemProtocol, (VOID **)&FileSystem);
    if (EFI_ERROR(Status) || FileSystem == NULL) return EFI_UNSUPPORTED;
    Status = uefi_call_wrapper(FileSystem->OpenVolume, 2, FileSystem, &Root);
    if (EFI_ERROR(Status) || Root == NULL) return Status;
    Status = uefi_call_wrapper(Root->Open, 5, Root, &Marker, Path,
                               EFI_FILE_MODE_READ | EFI_FILE_MODE_WRITE | EFI_FILE_MODE_CREATE,
                               0);
    if (!EFI_ERROR(Status) && Marker != NULL) {
        Status = uefi_call_wrapper(Marker->Write, 3, Marker, &PayloadSize, Payload);
        uefi_call_wrapper(Marker->Flush, 1, Marker);
        uefi_call_wrapper(Marker->Close, 1, Marker);
    }
    uefi_call_wrapper(Root->Close, 1, Root);
    return Status;
}
