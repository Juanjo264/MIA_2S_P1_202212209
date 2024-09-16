package FileSystem

import (
	"backend/DiskManagement"
	"backend/Structs"
	"backend/Utilities"
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

func Mkfs(id string, type_ string, fs_ string) (string, error) {
	// Variable para acumular los mensajes
	var logs string

	// Agregar los mensajes de inicio al log
	logs += "======INICIO MKFS======\n"
	logs += fmt.Sprintf("Id: %s\n", id)
	logs += fmt.Sprintf("Type: %s\n", type_)
	logs += fmt.Sprintf("Fs: %s\n", fs_)

	// Buscar la partición montada por ID
	var mountedPartition DiskManagement.MountedPartition
	var partitionFound bool

	for _, partitions := range DiskManagement.GetMountedPartitions() {
		for _, partition := range partitions {
			if partition.ID == id {
				mountedPartition = partition
				partitionFound = true
				break
			}
		}
		if partitionFound {
			break
		}
	}

	if !partitionFound {
		errMsg := "Partición no encontrada"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	if mountedPartition.Status != '1' { // Verifica si la partición está montada
		errMsg := "La partición aún no está montada"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Abrir archivo binario
	file, err := Utilities.OpenFile(mountedPartition.Path)
	if err != nil {
		errMsg := fmt.Sprintf("Error al abrir el archivo: %v", err)
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}
	defer file.Close()

	var TempMBR Structs.MBR
	// Leer objeto desde archivo binario
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		errMsg := "Error al leer MBR del archivo"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Formatear el MBR y agregarlo al log
	logs += fmt.Sprintf("MBR Size: %d\n", TempMBR.MbrSize)
	logs += fmt.Sprintf("MBR Signature: %d\n", TempMBR.Signature)
	logs += fmt.Sprintf("MBR Fit: %s\n", string(TempMBR.Fit[:]))
	logs += fmt.Sprintf("MBR Creation Date: %s\n", string(TempMBR.CreationDate[:]))
	logs += "-------------\n"

	var index int = -1
	// Iterar sobre las particiones para encontrar la que tiene el nombre correspondiente
	for i := 0; i < 4; i++ {
		if TempMBR.Partitions[i].Size != 0 {
			if strings.Contains(string(TempMBR.Partitions[i].Id[:]), id) {
				index = i
				break
			}
		}
	}

	if index != -1 {
		logs += fmt.Sprintf("Partición encontrada: %s\n", string(TempMBR.Partitions[index].Name[:]))
	} else {
		errMsg := "Partición no encontrada (2)"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	numerador := int32(TempMBR.Partitions[index].Size - int32(binary.Size(Structs.Superblock{})))
	denominador_base := int32(4 + int32(binary.Size(Structs.Inode{})) + 3*int32(binary.Size(Structs.Fileblock{})))
	var temp int32 = 0
	if fs_ == "2fs" {
		temp = 0
	} else {
		errMsg := "Error: por el momento solo está disponible 2FS."
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}
	denominador := denominador_base + temp
	n := int32(numerador / denominador)

	logs += fmt.Sprintf("INODOS: %d\n", n)

	// Crear el Superblock con todos los campos calculados
	var newSuperblock Structs.Superblock
	newSuperblock.S_filesystem_type = 2 // EXT2
	newSuperblock.S_inodes_count = n
	newSuperblock.S_blocks_count = 3 * n
	newSuperblock.S_free_blocks_count = 3*n - 2
	newSuperblock.S_free_inodes_count = n - 2
	copy(newSuperblock.S_mtime[:], "23/08/2024")
	copy(newSuperblock.S_umtime[:], "23/08/2024")
	newSuperblock.S_mnt_count = 1
	newSuperblock.S_magic = 0xEF53
	newSuperblock.S_inode_size = int32(binary.Size(Structs.Inode{}))
	newSuperblock.S_block_size = int32(binary.Size(Structs.Fileblock{}))

	// Calcula las posiciones de inicio
	newSuperblock.S_bm_inode_start = TempMBR.Partitions[index].Start + int32(binary.Size(Structs.Superblock{}))
	newSuperblock.S_bm_block_start = newSuperblock.S_bm_inode_start + n
	newSuperblock.S_inode_start = newSuperblock.S_bm_block_start + 3*n
	newSuperblock.S_block_start = newSuperblock.S_inode_start + n*newSuperblock.S_inode_size

	if fs_ == "2fs" {
		logs += "Creando EXT2...\n"
		create_ext2(n, TempMBR.Partitions[index], newSuperblock, "23/08/2024", file)
	} else {
		errMsg := "EXT3 no está soportado."
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	logs += "======FIN MKFS======\n"
	return logs + fmt.Sprintf("Formateo exitoso a: %s", id), nil
}

func create_ext2(n int32, partition Structs.Partition, newSuperblock Structs.Superblock, date string, file *os.File) {
	fmt.Println("======Start CREATE EXT2======")
	fmt.Println("INODOS:", n)

	// Imprimir Superblock inicial
	Structs.PrintSuperblock(newSuperblock)
	fmt.Println("Date:", date)

	// Escribe los bitmaps de inodos y bloques en el archivo
	for i := int32(0); i < n; i++ {
		if err := Utilities.WriteObject(file, byte(0), int64(newSuperblock.S_bm_inode_start+i)); err != nil {
			fmt.Println("Error: ", err)
			return
		}
	}

	for i := int32(0); i < 3*n; i++ {
		if err := Utilities.WriteObject(file, byte(0), int64(newSuperblock.S_bm_block_start+i)); err != nil {
			fmt.Println("Error: ", err)
			return
		}
	}

	// Inicializa inodos y bloques con valores predeterminados
	if err := initInodesAndBlocks(n, newSuperblock, file); err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// Crea la carpeta raíz y el archivo users.txt
	if err := createRootAndUsersFile(newSuperblock, date, file); err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// Escribe el superbloque actualizado al archivo
	if err := Utilities.WriteObject(file, newSuperblock, int64(partition.Start)); err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// Marca los primeros inodos y bloques como usados
	if err := markUsedInodesAndBlocks(newSuperblock, file); err != nil {
		fmt.Println("Error: ", err)
		return
	}

	// Leer e imprimir los inodos después de formatear
	fmt.Println("====== Imprimiendo Inodos ======")
	for i := int32(0); i < n; i++ {
		var inode Structs.Inode
		offset := int64(newSuperblock.S_inode_start + i*int32(binary.Size(Structs.Inode{})))
		if err := Utilities.ReadObject(file, &inode, offset); err != nil {
			fmt.Println("Error al leer inodo: ", err)
			return
		}

		// Verificar si el inodo está vacío (todos los I_block son -1)
		isEmpty := true
		for _, block := range inode.I_block {
			if block != -1 {
				isEmpty = false
				break
			}
		}

		// Imprimir solo si el inodo no está vacío
		if !isEmpty {
			Structs.PrintInode(inode)
		}
	}

	// Leer e imprimir los Folderblocks y Fileblocks después de formatear
	fmt.Println("====== Imprimiendo Folderblocks y Fileblocks ======")

	// Imprimir Folderblocks
	for i := int32(0); i < 1; i++ {
		var folderblock Structs.Folderblock
		offset := int64(newSuperblock.S_block_start + i*int32(binary.Size(Structs.Folderblock{})))
		if err := Utilities.ReadObject(file, &folderblock, offset); err != nil {
			fmt.Println("Error al leer Folderblock: ", err)
			return
		}
		Structs.PrintFolderblock(folderblock)
	}

	// Imprimir Fileblocks
	for i := int32(0); i < 1; i++ {
		var fileblock Structs.Fileblock
		offset := int64(newSuperblock.S_block_start + int32(binary.Size(Structs.Folderblock{})) + i*int32(binary.Size(Structs.Fileblock{})))
		if err := Utilities.ReadObject(file, &fileblock, offset); err != nil {
			fmt.Println("Error al leer Fileblock: ", err)
			return
		}
		Structs.PrintFileblock(fileblock)
	}

	// Imprimir el Superblock final
	Structs.PrintSuperblock(newSuperblock)

	fmt.Println("======End CREATE EXT2======")
}

// Función auxiliar para inicializar inodos y bloques
func initInodesAndBlocks(n int32, newSuperblock Structs.Superblock, file *os.File) error {
	var newInode Structs.Inode
	for i := int32(0); i < 15; i++ {
		newInode.I_block[i] = -1
	}

	for i := int32(0); i < n; i++ {
		if err := Utilities.WriteObject(file, newInode, int64(newSuperblock.S_inode_start+i*int32(binary.Size(Structs.Inode{})))); err != nil {
			return err
		}
	}

	var newFileblock Structs.Fileblock
	for i := int32(0); i < 3*n; i++ {
		if err := Utilities.WriteObject(file, newFileblock, int64(newSuperblock.S_block_start+i*int32(binary.Size(Structs.Fileblock{})))); err != nil {
			return err
		}
	}

	return nil
}

// Implementación de la función para obtener los inodos desde el sistema
func ObtenerInodosDesdeSistema(newSuperblock Structs.Superblock, file *os.File) ([]Structs.Inode, error) {
	var inodes []Structs.Inode
	n := newSuperblock.S_inodes_count

	// Leer cada inodo desde su posición calculada
	for i := int32(0); i < n; i++ {
		var inode Structs.Inode
		offset := int64(newSuperblock.S_inode_start + i*int32(binary.Size(Structs.Inode{})))
		if err := Utilities.ReadObject(file, &inode, offset); err != nil {
			return nil, fmt.Errorf("error al leer inodo: %v", err)
		}
		inodes = append(inodes, inode)
	}

	return inodes, nil
}

// Función auxiliar para crear la carpeta raíz y el archivo users.txt
func createRootAndUsersFile(newSuperblock Structs.Superblock, date string, file *os.File) error {
	var Inode0, Inode1 Structs.Inode
	initInode(&Inode0, date)
	initInode(&Inode1, date)

	Inode0.I_block[0] = 0
	Inode1.I_block[0] = 1

	// Asignar el tamaño real del contenido
	data := "1,G,root\n1,U,root,root,123\n"
	actualSize := int32(len(data))
	Inode1.I_size = actualSize // Esto ahora refleja el tamaño real del contenido

	var Fileblock1 Structs.Fileblock
	copy(Fileblock1.B_content[:], data) // Copia segura de datos a Fileblock

	var Folderblock0 Structs.Folderblock
	Folderblock0.B_content[0].B_inodo = 0
	copy(Folderblock0.B_content[0].B_name[:], ".")
	Folderblock0.B_content[1].B_inodo = 0
	copy(Folderblock0.B_content[1].B_name[:], "..")
	Folderblock0.B_content[2].B_inodo = 1
	copy(Folderblock0.B_content[2].B_name[:], "users.txt")

	// Escribir los inodos y bloques en las posiciones correctas
	if err := Utilities.WriteObject(file, Inode0, int64(newSuperblock.S_inode_start)); err != nil {
		return err
	}
	if err := Utilities.WriteObject(file, Inode1, int64(newSuperblock.S_inode_start+int32(binary.Size(Structs.Inode{})))); err != nil {
		return err
	}
	if err := Utilities.WriteObject(file, Folderblock0, int64(newSuperblock.S_block_start)); err != nil {
		return err
	}
	if err := Utilities.WriteObject(file, Fileblock1, int64(newSuperblock.S_block_start+int32(binary.Size(Structs.Folderblock{})))); err != nil {
		return err
	}

	return nil
}

// Función auxiliar para inicializar un inodo
func initInode(inode *Structs.Inode, date string) {
	inode.I_uid = 1
	inode.I_gid = 1
	inode.I_size = 0
	copy(inode.I_atime[:], date)
	copy(inode.I_ctime[:], date)
	copy(inode.I_mtime[:], date)
	copy(inode.I_perm[:], "664")

	for i := int32(0); i < 15; i++ {
		inode.I_block[i] = -1
	}
}

// Función auxiliar para marcar los inodos y bloques usados
func markUsedInodesAndBlocks(newSuperblock Structs.Superblock, file *os.File) error {
	if err := Utilities.WriteObject(file, byte(1), int64(newSuperblock.S_bm_inode_start)); err != nil {
		return err
	}
	if err := Utilities.WriteObject(file, byte(1), int64(newSuperblock.S_bm_inode_start+1)); err != nil {
		return err
	}
	if err := Utilities.WriteObject(file, byte(1), int64(newSuperblock.S_bm_block_start)); err != nil {
		return err
	}
	if err := Utilities.WriteObject(file, byte(1), int64(newSuperblock.S_bm_block_start+1)); err != nil {
		return err
	}
	return nil
}

// Función para crear un nuevo directorio dentro de un sistema de archivos
func CreateDirectory(superblock Structs.Superblock, path string, date string, file *os.File) error {
	// Split the path into components
	components := strings.Split(path, "/")
	components = components[1:] // Remove the empty string before the first '/'

	currentInodeIndex := int32(0) // Start from the root directory

	for i, dirName := range components {
		var currentInode Structs.Inode
		inodePos := int64(superblock.S_inode_start + currentInodeIndex*int32(binary.Size(Structs.Inode{})))
		if err := Utilities.ReadObject(file, &currentInode, inodePos); err != nil {
			return fmt.Errorf("error reading inode: %v", err)
		}

		// Check if the directory already exists
		existingInodeIndex, err := FindDirectoryEntry(superblock, currentInode, dirName, file)
		if err != nil {
			return err
		}

		if existingInodeIndex != -1 {
			// Directory already exists, move to the next component
			currentInodeIndex = existingInodeIndex
			continue
		}

		// If this is the last component or the directory doesn't exist, create it
		if i == len(components)-1 || existingInodeIndex == -1 {
			// Initialize a new inode for the directory
			var newInode Structs.Inode
			initInode(&newInode, date)
			newInode.I_block[0] = findFreeBlock(superblock, file)
			if newInode.I_block[0] == -1 {
				return fmt.Errorf("no free blocks available")
			}
			newInode.I_size = 0
			newInode.I_type = [1]byte{1} // Indicates it's a directory

			// Create a Folderblock for the new directory
			var newFolderblock Structs.Folderblock
			newFolderblock.B_content[0].B_inodo = currentInodeIndex
			copy(newFolderblock.B_content[0].B_name[:], "..")
			newFolderblock.B_content[1].B_inodo = findFreeInode(superblock, file)
			if newFolderblock.B_content[1].B_inodo == -1 {
				return fmt.Errorf("no free inodes available")
			}
			copy(newFolderblock.B_content[1].B_name[:], ".")

			// Write the new inode and Folderblock
			newInodePos := int64(superblock.S_inode_start + newFolderblock.B_content[1].B_inodo*int32(binary.Size(Structs.Inode{})))
			if err := Utilities.WriteObject(file, newInode, newInodePos); err != nil {
				return fmt.Errorf("error writing new inode: %v", err)
			}

			blockPos := int64(superblock.S_block_start + newInode.I_block[0]*int32(binary.Size(Structs.Folderblock{})))
			if err := Utilities.WriteObject(file, newFolderblock, blockPos); err != nil {
				return fmt.Errorf("error writing new folder block: %v", err)
			}

			// Update parent directory
			if err := updateParentDirectory(superblock, currentInode, currentInodeIndex, newFolderblock.B_content[1].B_inodo, dirName, file); err != nil {
				return err
			}

			// Mark inode and block as used
			if err := markUsedInodeAndBlock(superblock, newInode, newInode.I_block[0], file); err != nil {
				return fmt.Errorf("error updating bitmaps: %v", err)
			}

			currentInodeIndex = newFolderblock.B_content[1].B_inodo
		}
	}

	fmt.Printf("Directory '%s' created successfully.\n", path)
	return nil
}

func FindDirectoryEntry(superblock Structs.Superblock, inode Structs.Inode, name string, file *os.File) (int32, error) {
	for i := 0; i < 15; i++ {
		if inode.I_block[i] != -1 {
			var folderblock Structs.Folderblock
			blockPos := int64(superblock.S_block_start + inode.I_block[i]*int32(binary.Size(Structs.Folderblock{})))
			if err := Utilities.ReadObject(file, &folderblock, blockPos); err != nil {
				return -1, fmt.Errorf("error reading folder block: %v", err)
			}

			for _, entry := range folderblock.B_content {
				entryName := string(bytes.Trim(entry.B_name[:], "\x00"))
				if entryName == name {
					return entry.B_inodo, nil
				}
			}
		}
	}
	return -1, nil
}

func updateParentDirectory(superblock Structs.Superblock, parentInode Structs.Inode, parentInodeIndex int32, newInodeIndex int32, dirName string, file *os.File) error {
	for i := 0; i < 15; i++ {
		if parentInode.I_block[i] != -1 {
			var folderblock Structs.Folderblock
			blockPos := int64(superblock.S_block_start + parentInode.I_block[i]*int32(binary.Size(Structs.Folderblock{})))
			if err := Utilities.ReadObject(file, &folderblock, blockPos); err != nil {
				return fmt.Errorf("error reading parent folder block: %v", err)
			}

			for j := 0; j < 4; j++ {
				if folderblock.B_content[j].B_inodo == 0 {
					folderblock.B_content[j].B_inodo = newInodeIndex
					copy(folderblock.B_content[j].B_name[:], dirName)

					if err := Utilities.WriteObject(file, folderblock, blockPos); err != nil {
						return fmt.Errorf("error writing updated parent folder block: %v", err)
					}

					return nil
				}
			}
		}
	}

	return fmt.Errorf("no space in parent directory to add new directory entry")
}

// Función auxiliar para encontrar un bloque libre
func findFreeBlock(superblock Structs.Superblock, file *os.File) int32 {
	// Este código debe recorrer el bitmap de bloques en busca de un bloque libre
	// Retorna el índice del primer bloque libre encontrado
	var block byte
	for i := int32(0); i < superblock.S_blocks_count; i++ {
		pos := int64(superblock.S_bm_block_start + i)
		if err := Utilities.ReadObject(file, &block, pos); err != nil {
			continue
		}
		if block == 0 {
			return i
		}
	}
	return -1 // No se encontró un bloque libre
}

// Función auxiliar para marcar un inodo y bloque como usados
func markUsedInodeAndBlock(superblock Structs.Superblock, inode Structs.Inode, blockIndex int32, file *os.File) error {
	// Marca el bitmap de inodos
	if err := Utilities.WriteObject(file, byte(1), int64(superblock.S_bm_inode_start+inode.I_block[0])); err != nil {
		return err
	}
	// Marca el bitmap de bloques
	if err := Utilities.WriteObject(file, byte(1), int64(superblock.S_bm_block_start+blockIndex)); err != nil {
		return err
	}
	return nil
}

func CreateFile(superblock Structs.Superblock, parentInodeIndex int32, fileName string, size int32, content string, date string, file *os.File) error {
	// Initialize a new inode for the file
	var newInode Structs.Inode
	initInode(&newInode, date)
	newInode.I_size = size
	newInode.I_type = [1]byte{0} // Indicates it's a file

	// If content is empty, generate default content
	if content == "" {
		content = generateContent(int(size))
	}

	// Split content into chunks
	chunks := Utilities.SplitStringIntoChunks(content)

	// Allocate blocks for the file
	blocksNeeded := int32(len(chunks))
	for i := int32(0); i < blocksNeeded && i < 15; i++ {
		newInode.I_block[i] = findFreeBlock(superblock, file)
		if newInode.I_block[i] == -1 {
			return fmt.Errorf("no free blocks available")
		}

		// Create a Fileblock with the chunk content
		var newFileblock Structs.Fileblock
		copy(newFileblock.B_content[:], chunks[i])

		// Write the Fileblock to disk
		blockPos := int64(superblock.S_block_start + newInode.I_block[i]*int32(binary.Size(Structs.Fileblock{})))
		if err := Utilities.WriteObject(file, newFileblock, blockPos); err != nil {
			return fmt.Errorf("error writing new file block: %v", err)
		}

		// Mark the block as used in the bitmap
		if err := Utilities.WriteObject(file, byte(1), int64(superblock.S_bm_block_start+newInode.I_block[i])); err != nil {
			return fmt.Errorf("error updating block bitmap: %v", err)
		}
	}

	// Find a free inode
	newInodeIndex := findFreeInode(superblock, file)
	if newInodeIndex == -1 {
		return fmt.Errorf("no free inodes available")
	}

	// Write the new inode
	inodePos := int64(superblock.S_inode_start + newInodeIndex*int32(binary.Size(Structs.Inode{})))
	if err := Utilities.WriteObject(file, newInode, inodePos); err != nil {
		return fmt.Errorf("error writing new inode: %v", err)
	}

	// Mark the inode as used in the bitmap
	if err := Utilities.WriteObject(file, byte(1), int64(superblock.S_bm_inode_start+newInodeIndex)); err != nil {
		return fmt.Errorf("error updating inode bitmap: %v", err)
	}

	// Update parent directory
	var parentInode Structs.Inode
	parentInodePos := int64(superblock.S_inode_start + parentInodeIndex*int32(binary.Size(Structs.Inode{})))
	if err := Utilities.ReadObject(file, &parentInode, parentInodePos); err != nil {
		return fmt.Errorf("error reading parent inode: %v", err)
	}

	// Find the first non-empty block in the parent directory
	var parentFolderblock Structs.Folderblock
	var blockIndex int32
	for blockIndex = 0; blockIndex < 15; blockIndex++ {
		if parentInode.I_block[blockIndex] != -1 {
			blockPos := int64(superblock.S_block_start + parentInode.I_block[blockIndex]*int32(binary.Size(Structs.Folderblock{})))
			if err := Utilities.ReadObject(file, &parentFolderblock, blockPos); err != nil {
				return fmt.Errorf("error reading parent folder block: %v", err)
			}

			// Find an empty slot in the folder block
			for i := 0; i < 4; i++ {
				if parentFolderblock.B_content[i].B_inodo == 0 {
					parentFolderblock.B_content[i].B_inodo = newInodeIndex
					copy(parentFolderblock.B_content[i].B_name[:], fileName)

					// Write the updated folder block back to disk
					if err := Utilities.WriteObject(file, parentFolderblock, blockPos); err != nil {
						return fmt.Errorf("error writing updated parent folder block: %v", err)
					}

					fmt.Printf("File '%s' created successfully.\n", fileName)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("no space in parent directory to add new file entry")
}
func generateContent(size int) string {
	content := ""
	for len(content) < size {
		content += "0123456789"
	}
	return content[:size] // Truncate the string to the exact size
}
func findFreeInode(superblock Structs.Superblock, file *os.File) int32 {
	var inodeBit byte
	for i := int32(0); i < superblock.S_inodes_count; i++ {
		pos := int64(superblock.S_bm_inode_start + i)
		if err := Utilities.ReadObject(file, &inodeBit, pos); err != nil {
			continue
		}
		if inodeBit == 0 {
			return i
		}
	}
	return -1 // No free inode found
}

// ListDirectories lista los directorios y archivos desde un inodo dado
func ListDirectories(superblock Structs.Superblock, inodeIndex int32, path string, file *os.File) error {
	// Leer el inodo desde su posición
	var inode Structs.Inode
	inodePos := int64(superblock.S_inode_start + inodeIndex*int32(binary.Size(Structs.Inode{})))
	if err := Utilities.ReadObject(file, &inode, inodePos); err != nil {
		return fmt.Errorf("error al leer inodo: %v", err)
	}

	// Verificar si el inodo es un directorio
	if inode.I_type[0] != 1 {
		return fmt.Errorf("el inodo en %s no es un directorio", path)
	}

	fmt.Printf("Contenido de %s:\n", path)
	entriesFound := false // Variable para verificar si hay entradas encontradas

	// Recorrer los bloques de carpetas asignados al inodo
	for i := 0; i < 15; i++ {
		if inode.I_block[i] != -1 {
			// Leer el bloque de carpeta
			var folderBlock Structs.Folderblock
			blockPos := int64(superblock.S_block_start + inode.I_block[i]*int32(binary.Size(Structs.Folderblock{})))
			if err := Utilities.ReadObject(file, &folderBlock, blockPos); err != nil {
				return fmt.Errorf("error al leer Folderblock: %v", err)
			}

			// Mostrar los contenidos del Folderblock
			for _, entry := range folderBlock.B_content {
				entryName := strings.TrimSpace(string(entry.B_name[:]))
				if entryName != "" && entryName != "." && entryName != ".." {
					fmt.Printf("- %s\n", entryName)
					entriesFound = true // Se encontró al menos una entrada

					// Si es un directorio, listar recursivamente su contenido
					if entry.B_inodo != 0 {
						newPath := path + "/" + entryName
						if err := ListDirectories(superblock, entry.B_inodo, newPath, file); err != nil {
							fmt.Printf("Error listando %s: %v\n", newPath, err)
						}
					}
				}
			}
		}
	}

	// Si no se encontraron entradas, indicar que el directorio está vacío
	if !entriesFound {
		fmt.Printf("El directorio '%s' está vacío.\n", path)
	}

	return nil
}
