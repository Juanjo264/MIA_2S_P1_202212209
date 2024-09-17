package FileSystem

import (
	"backend/DiskManagement"
	"backend/Structs"
	"backend/User"
	"backend/Utilities"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Función Mkfs que inicializa el sistema de archivos
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

	// Obtener la fecha actual formateada como "DD/MM/YYYY"
	currentDate := time.Now().Format("02/01/2006")

	// Crear el Superblock con todos los campos calculados
	var newSuperblock Structs.Superblock
	newSuperblock.S_filesystem_type = 2 // EXT2
	newSuperblock.S_inodes_count = n
	newSuperblock.S_blocks_count = 3 * n
	newSuperblock.S_free_blocks_count = 3*n - 2
	newSuperblock.S_free_inodes_count = n - 2
	copy(newSuperblock.S_mtime[:], currentDate)
	copy(newSuperblock.S_umtime[:], currentDate)
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
		create_ext2(n, TempMBR.Partitions[index], newSuperblock, currentDate, file)
	} else {
		errMsg := "EXT3 no está soportado."
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	logs += "======FIN MKFS======\n"
	return logs + fmt.Sprintf("Formateo exitoso a: %s", id), nil
}

// Función create_ext2 que utiliza la fecha actual para crear EXT2
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
	printInodes(n, newSuperblock, file)
	printBlocks(newSuperblock, file)

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
	initInode(&Inode0)
	initInode(&Inode1)

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
func initInode(inode *Structs.Inode) {
	// Obtener la fecha actual y formatearla como "DD/MM/YYYY"
	currentDate := time.Now().Format("02/01/2006")

	inode.I_uid = 1
	inode.I_gid = 1
	inode.I_size = 0
	copy(inode.I_atime[:], currentDate)
	copy(inode.I_ctime[:], currentDate)
	copy(inode.I_mtime[:], currentDate)
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

func printInodes(n int32, newSuperblock Structs.Superblock, file *os.File) {
	fmt.Println("====== Imprimiendo Inodos ======")
	for i := int32(0); i < n; i++ {
		var inode Structs.Inode
		offset := int64(newSuperblock.S_inode_start + i*int32(binary.Size(Structs.Inode{})))
		if err := Utilities.ReadObject(file, &inode, offset); err != nil {
			fmt.Println("Error al leer inodo: ", err)
			return
		}

		isEmpty := true
		for _, block := range inode.I_block {
			if block != -1 {
				isEmpty = false
				break
			}
		}

		if !isEmpty {
			Structs.PrintInode(inode)
		}
	}
}
func printBlocks(newSuperblock Structs.Superblock, file *os.File) {
	fmt.Println("====== Imprimiendo Folderblocks y Fileblocks ======")

	for i := int32(0); i < 1; i++ {
		var folderblock Structs.Folderblock
		offset := int64(newSuperblock.S_block_start + i*int32(binary.Size(Structs.Folderblock{})))
		if err := Utilities.ReadObject(file, &folderblock, offset); err != nil {
			fmt.Println("Error al leer Folderblock: ", err)
			return
		}
		Structs.PrintFolderblock(folderblock)
	}

	for i := int32(0); i < 1; i++ {
		var fileblock Structs.Fileblock
		offset := int64(newSuperblock.S_block_start + int32(binary.Size(Structs.Folderblock{})) + i*int32(binary.Size(Structs.Fileblock{})))
		if err := Utilities.ReadObject(file, &fileblock, offset); err != nil {
			fmt.Println("Error al leer Fileblock: ", err)
			return
		}
		Structs.PrintFileblock(fileblock)
	}
}

// Mkdir crea directorios de manera jerárquica en el sistema de archivos
func Mkdir(path string) (string, error) {
	// Variable para acumular los mensajes
	var logs string

	// Agregar los mensajes de inicio al log
	logs += "======INICIO MKDIR======\n"
	logs += fmt.Sprintf("Path: %s\n", path)

	// Verificar si hay una partición logueada
	if User.CurrentLoggedPartitionID == "" {
		errMsg := "No hay ninguna partición logueada"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Buscar la partición montada por ID
	var mountedPartition DiskManagement.MountedPartition
	var partitionFound bool

	for _, partitions := range DiskManagement.GetMountedPartitions() {
		for _, partition := range partitions {
			if partition.ID == User.CurrentLoggedPartitionID {
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
			if strings.Contains(string(TempMBR.Partitions[i].Id[:]), User.CurrentLoggedPartitionID) {
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

	// Leer el superbloque
	var superblock Structs.Superblock
	if err := Utilities.ReadObject(file, &superblock, int64(TempMBR.Partitions[index].Start)); err != nil {
		errMsg := "Error al leer el superbloque"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Crear los directorios
	directories := strings.Split(path, "/")
	currentInode := int32(0) // Asumimos que el inodo raíz es 0

	for _, dir := range directories {
		if dir == "" {
			continue
		}

		// Busca si el directorio ya existe
		found, inodeIndex := findDirectory(dir, currentInode, file, superblock)
		if found {
			currentInode = inodeIndex
		} else {
			// Crea el nuevo directorio
			newInodeIndex, err := createDirectory(dir, currentInode, file, superblock)
			if err != nil {
				return logs, err
			}
			currentInode = newInodeIndex
		}
	}

	logs += "======FIN MKDIR======\n"
	return logs + fmt.Sprintf("Directorio creado: %s", path), nil
}

func findDirectory(name string, parentInode int32, file *os.File, superblock Structs.Superblock) (bool, int32) {
	var inode Structs.Inode
	offset := int64(superblock.S_inode_start + parentInode*int32(binary.Size(Structs.Inode{})))
	if err := Utilities.ReadObject(file, &inode, offset); err != nil {
		return false, -1
	}

	for _, block := range inode.I_block {
		if block == -1 {
			continue
		}

		var folderblock Structs.Folderblock
		offset := int64(superblock.S_block_start + block*int32(binary.Size(Structs.Folderblock{})))
		if err := Utilities.ReadObject(file, &folderblock, offset); err != nil {
			return false, -1
		}

		for _, content := range folderblock.B_content {
			if string(content.B_name[:]) == name {
				return true, content.B_inodo
			}
		}
	}

	return false, -1
}

// Función para crear un nuevo directorio
func createDirectory(name string, parentInode int32, file *os.File, superblock Structs.Superblock) (int32, error) {
	// Encuentra un inodo libre
	newInodeIndex := findFreeInode(superblock, file)
	if newInodeIndex == -1 {
		return -1, fmt.Errorf("no hay inodos libres")
	}

	// Encuentra un bloque libre
	newBlockIndex := findFreeBlock(superblock, file)
	if newBlockIndex == -1 {
		return -1, fmt.Errorf("no hay bloques libres")
	}

	// Inicializa el nuevo inodo con valores predeterminados
	var newInode Structs.Inode
	initInode(&newInode)
	newInode.I_block[0] = newBlockIndex

	// Escribe el nuevo inodo en el archivo en la posición correcta
	inodeOffset := int64(superblock.S_inode_start + newInodeIndex*int32(binary.Size(Structs.Inode{})))
	if err := Utilities.WriteObject(file, newInode, inodeOffset); err != nil {
		return -1, fmt.Errorf("error al escribir el nuevo inodo: %v", err)
	}

	// Inicializa el nuevo folderblock y escribe en el archivo
	var newFolderblock Structs.Folderblock
	newFolderblock.B_content[0].B_inodo = newInodeIndex
	copy(newFolderblock.B_content[0].B_name[:], name)

	blockOffset := int64(superblock.S_block_start + newBlockIndex*int32(binary.Size(Structs.Folderblock{})))
	if err := Utilities.WriteObject(file, newFolderblock, blockOffset); err != nil {
		return -1, fmt.Errorf("error al escribir el folderblock: %v", err)
	}

	// Marca el inodo y el bloque como usados en los bitmaps
	if err := markInodeAsUsed(newInodeIndex, superblock, file); err != nil {
		return -1, fmt.Errorf("error al marcar el inodo como usado: %v", err)
	}
	if err := markBlockAsUsed(newBlockIndex, superblock, file); err != nil {
		return -1, fmt.Errorf("error al marcar el bloque como usado: %v", err)
	}

	// Actualiza el folderblock del inodo padre
	if err := updateParentFolderblock(name, parentInode, newInodeIndex, file, superblock); err != nil {
		return -1, fmt.Errorf("error al actualizar el folderblock del inodo padre: %v", err)
	}

	return newInodeIndex, nil
}

func findFreeInode(superblock Structs.Superblock, file *os.File) int32 {
	for i := int32(0); i < superblock.S_inodes_count; i++ {
		var status byte
		offset := int64(superblock.S_bm_inode_start + i)
		if err := Utilities.ReadObject(file, &status, offset); err != nil {
			return -1
		}
		if status == 0 {
			return i
		}
	}
	return -1
}

func findFreeBlock(superblock Structs.Superblock, file *os.File) int32 {
	for i := int32(0); i < superblock.S_blocks_count; i++ {
		var status byte
		offset := int64(superblock.S_bm_block_start + i)
		if err := Utilities.ReadObject(file, &status, offset); err != nil {
			return -1
		}
		if status == 0 {
			return i
		}
	}
	return -1
}

func updateParentFolderblock(name string, parentInode int32, newInodeIndex int32, file *os.File, superblock Structs.Superblock) error {
	var inode Structs.Inode
	inodeOffset := int64(superblock.S_inode_start + parentInode*int32(binary.Size(Structs.Inode{})))
	if err := Utilities.ReadObject(file, &inode, inodeOffset); err != nil {
		return fmt.Errorf("error al leer el inodo padre: %v", err)
	}

	for i, block := range inode.I_block {
		if block == -1 {
			// Si no hay un bloque asignado, asignar uno nuevo
			newBlockIndex := findFreeBlock(superblock, file)
			if newBlockIndex == -1 {
				return fmt.Errorf("no hay bloques libres")
			}
			inode.I_block[i] = newBlockIndex

			// Inicializar el nuevo folderblock y escribirlo
			var newFolderblock Structs.Folderblock
			newFolderblock.B_content[0].B_inodo = newInodeIndex
			copy(newFolderblock.B_content[0].B_name[:], name)

			blockOffset := int64(superblock.S_block_start + newBlockIndex*int32(binary.Size(Structs.Folderblock{})))
			if err := Utilities.WriteObject(file, newFolderblock, blockOffset); err != nil {
				return fmt.Errorf("error al escribir el nuevo folderblock: %v", err)
			}

			// Escribir el inodo actualizado en el archivo
			if err := Utilities.WriteObject(file, inode, inodeOffset); err != nil {
				return fmt.Errorf("error al actualizar el inodo padre: %v", err)
			}

			// Marca el bloque como usado en el bitmap
			if err := markBlockAsUsed(newBlockIndex, superblock, file); err != nil {
				return fmt.Errorf("error al marcar el bloque como usado: %v", err)
			}

			return nil
		}

		// Leer el folderblock existente
		var folderblock Structs.Folderblock
		blockOffset := int64(superblock.S_block_start + block*int32(binary.Size(Structs.Folderblock{})))
		if err := Utilities.ReadObject(file, &folderblock, blockOffset); err != nil {
			return fmt.Errorf("error al leer el folderblock existente: %v", err)
		}

		// Buscar un espacio vacío dentro del folderblock
		for j, content := range folderblock.B_content {
			if content.B_inodo == -1 {
				folderblock.B_content[j].B_inodo = newInodeIndex
				copy(folderblock.B_content[j].B_name[:], name)
				if err := Utilities.WriteObject(file, folderblock, blockOffset); err != nil {
					return fmt.Errorf("error al actualizar el folderblock: %v", err)
				}
				return nil
			}
		}
	}

	// Si no se encontró espacio, lanza un error
	return fmt.Errorf("no hay espacio en el folderblock del inodo padre")
}

func handleIndirectBlocks(blockIndex int32, name string, newInodeIndex int32, file *os.File, superblock Structs.Superblock) error {
	var pointerblock Structs.Pointerblock
	offset := int64(superblock.S_block_start + blockIndex*int32(binary.Size(Structs.Pointerblock{})))
	if err := Utilities.ReadObject(file, &pointerblock, offset); err != nil {
		return err
	}

	for i, pointer := range pointerblock.B_pointers {
		if pointer == -1 {
			// Asignar un nuevo bloque
			newBlockIndex := findFreeBlock(superblock, file)
			if newBlockIndex == -1 {
				return fmt.Errorf("no hay bloques libres")
			}
			pointerblock.B_pointers[i] = newBlockIndex

			// Inicializar el nuevo folderblock
			var newFolderblock Structs.Folderblock
			newFolderblock.B_content[0].B_inodo = newInodeIndex
			copy(newFolderblock.B_content[0].B_name[:], name)

			// Escribir el nuevo folderblock en el archivo
			offset = int64(superblock.S_block_start + newBlockIndex*int32(binary.Size(Structs.Folderblock{})))
			if err := Utilities.WriteObject(file, newFolderblock, offset); err != nil {
				return err
			}

			// Escribir el pointerblock actualizado en el archivo
			offset = int64(superblock.S_block_start + blockIndex*int32(binary.Size(Structs.Pointerblock{})))
			if err := Utilities.WriteObject(file, pointerblock, offset); err != nil {
				return err
			}

			return nil
		}

		var folderblock Structs.Folderblock
		offset = int64(superblock.S_block_start + pointer*int32(binary.Size(Structs.Folderblock{})))
		if err := Utilities.ReadObject(file, &folderblock, offset); err != nil {
			return err
		}

		for j, content := range folderblock.B_content {
			if content.B_inodo == -1 {
				folderblock.B_content[j].B_inodo = newInodeIndex
				copy(folderblock.B_content[j].B_name[:], name)
				if err := Utilities.WriteObject(file, folderblock, offset); err != nil {
					return err
				}
				return nil
			}
		}
	}

	return fmt.Errorf("no hay espacio en los bloques indirectos")
}

type MKFILE struct {
	path string // Ruta del archivo
	r    bool   // Opción recursiva
	size int    // Tamaño del archivo
	cont string // Contenido del archivo
}

// ParserMkfile parsea el comando mkfile y devuelve una instancia de MKFILE
func ParserMkfile(tokens []string) (string, error) {
	cmd := &MKFILE{} // Crea una nueva instancia de MKFILE

	// Unir tokens en una sola cadena y luego dividir por espacios, respetando las comillas
	args := strings.Join(tokens, " ")
	// Expresión regular para encontrar los parámetros del comando mkfile
	re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-r|-size=\d+|-cont="[^"]+"|-cont=[^\s]+`)
	// Encuentra todas las coincidencias de la expresión regular en la cadena de argumentos
	matches := re.FindAllString(args, -1)

	// Verificar que todos los tokens fueron reconocidos por la expresión regular
	if len(matches) != len(tokens) {
		// Identificar el parámetro inválido
		for _, token := range tokens {
			if !re.MatchString(token) {
				return "", fmt.Errorf("parámetro inválido: %s", token)
			}
		}
	}

	// Itera sobre cada coincidencia encontrada
	for _, match := range matches {
		// Divide cada parte en clave y valor usando "=" como delimitador
		kv := strings.SplitN(match, "=", 2)
		key := strings.ToLower(kv[0])
		var value string
		if len(kv) == 2 {
			value = kv[1]
		}

		// Remove quotes from value if present
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		// Switch para manejar diferentes parámetros
		switch key {
		case "-path":
			// Verifica que el path no esté vacío
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			cmd.path = value
		case "-r":
			// Establece el valor de r a true
			cmd.r = true
		case "-size":
			// Convierte el valor del tamaño a un entero
			size, err := strconv.Atoi(value)
			if err != nil || size < 0 {
				return "", errors.New("el tamaño debe ser un número entero no negativo")
			}
			cmd.size = size
		case "-cont":
			// Verifica que el contenido no esté vacío
			if value == "" {
				return "", errors.New("el contenido no puede estar vacío")
			}
			cmd.cont = value
		default:
			// Si el parámetro no es reconocido, devuelve un error
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	// Verifica que el parámetro -path haya sido proporcionado
	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}

	// Si no se proporcionó el tamaño, se establece por defecto a 0
	if cmd.size == 0 {
		cmd.size = 0
	}

	// Si no se proporcionó el contenido, se establece por defecto a ""
	if cmd.cont == "" {
		cmd.cont = ""
	}

	// Crear el archivo con los parámetros proporcionados
	err := commandMkfile(cmd)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("MKFILE: Archivo %s creado correctamente.", cmd.path), nil // Devuelve el comando MKFILE creado
}

// Función para crear el archivo
func commandMkfile(mkfile *MKFILE) error {
	// Obtener la partición montada
	partition, err := getMountedPartition()
	if err != nil {
		return fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	// Generar el contenido del archivo si no se proporcionó
	if mkfile.cont == "" {
		mkfile.cont = generateContent(mkfile.size)
	}

	// Crear el archivo
	err = createFile(mkfile.path, mkfile.size, mkfile.cont, partition)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %w", err)
	}

	return nil
}

// generateContent genera una cadena de números del 0 al 9 hasta cumplir el tamaño ingresado
func generateContent(size int) string {
	content := ""
	for len(content) < size {
		content += "0123456789"
	}
	return content[:size] // Recorta la cadena al tamaño exacto
}

// Función para obtener la partición montada
func getMountedPartition() (*DiskManagement.MountedPartition, error) {
	if User.CurrentLoggedPartitionID == "" {
		return nil, errors.New("no hay ninguna partición logueada")
	}

	for _, partitions := range DiskManagement.GetMountedPartitions() {
		for _, partition := range partitions {
			if partition.ID == User.CurrentLoggedPartitionID {
				if partition.Status != '1' {
					return nil, errors.New("la partición aún no está montada")
				}
				return &partition, nil
			}
		}
	}

	return nil, errors.New("partición no encontrada")
}

// Función para crear un archivo
func createFile(filePath string, size int, content string, partition *DiskManagement.MountedPartition) error {
	// Abrir el archivo binario de la partición
	file, err := Utilities.OpenFile(partition.Path)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo: %v", err)
	}
	defer file.Close()

	// Leer el MBR para obtener el inicio de la partición
	var mbr Structs.MBR
	if err := Utilities.ReadObject(file, &mbr, 0); err != nil {
		return fmt.Errorf("error al leer el MBR: %v", err)
	}

	// Encontrar la partición montada
	var partitionStart int64
	for _, part := range mbr.Partitions {
		if strings.TrimSpace(string(part.Id[:])) == partition.ID {
			partitionStart = int64(part.Start)
			break
		}
	}

	if partitionStart == 0 {
		return fmt.Errorf("no se encontró la partición montada")
	}

	// Leer el superbloque
	var superblock Structs.Superblock
	if err := Utilities.ReadObject(file, &superblock, partitionStart); err != nil {
		return fmt.Errorf("error al leer el superbloque: %v", err)
	}

	// Crear los directorios padres si no existen
	parentDirs, destFile := getParentDirectories(filePath)
	currentInode := int32(0) // Asumimos que el inodo raíz es 0

	for _, dir := range parentDirs {
		if dir == "" {
			continue
		}

		// Busca si el directorio ya existe
		found, inodeIndex := findDirectory(dir, currentInode, file, superblock)
		if found {
			currentInode = inodeIndex
		} else {
			// Crea el nuevo directorio
			newInodeIndex, err := createDirectory(dir, currentInode, file, superblock)
			if err != nil {
				return err
			}
			currentInode = newInodeIndex
		}
	}

	// Crear el archivo en el directorio destino
	err = createFileInDirectory(destFile, currentInode, size, content, file, superblock, *partition)
	if err != nil {
		return fmt.Errorf("error al crear el archivo en el directorio destino: %w", err)
	}

	return nil
}

// Función para obtener los directorios padres y el archivo destino
func getParentDirectories(filePath string) ([]string, string) {
	dirs := strings.Split(filePath, "/")
	return dirs[:len(dirs)-1], dirs[len(dirs)-1]
}

// Función para crear un archivo en un directorio
func createFileInDirectory(fileName string, parentInode int32, size int, content string, file *os.File, superblock Structs.Superblock, mountedPartition DiskManagement.MountedPartition) error {
	// Leer el MBR para obtener el inicio de la partición
	var mbr Structs.MBR
	if err := Utilities.ReadObject(file, &mbr, 0); err != nil {
		return fmt.Errorf("error al leer el MBR: %v", err)
	}

	// Encontrar la partición montada y obtener su inicio
	var partitionStart int64
	var partitionIndex int = -1
	for i := 0; i < 4; i++ {
		if mbr.Partitions[i].Size != 0 && strings.Contains(string(mbr.Partitions[i].Id[:]), mountedPartition.ID) {
			partitionStart = int64(mbr.Partitions[i].Start)
			partitionIndex = i
			break
		}
	}

	if partitionStart == 0 {
		return fmt.Errorf("no se encontró la partición montada")
	}

	// Obtener la partición y verificar que haya sido encontrada
	if partitionIndex == -1 {
		return fmt.Errorf("no se encontró la partición correspondiente")
	}

	// Actualiza el superbloque si hay cambios en los inodos o bloques
	if err := Utilities.WriteObject(file, superblock, partitionStart); err != nil {
		return fmt.Errorf("error al actualizar el superbloque: %v", err)
	}

	// Encuentra un inodo libre
	newInodeIndex := findFreeInode(superblock, file)
	if newInodeIndex == -1 {
		return fmt.Errorf("no hay inodos libres")
	}

	// Encuentra un bloque libre
	newBlockIndex := findFreeBlock(superblock, file)
	if newBlockIndex == -1 {
		return fmt.Errorf("no hay bloques libres")
	}

	// Inicializa el nuevo inodo correctamente
	var newInode Structs.Inode
	initInode(&newInode) // Inicializa el inodo con valores predeterminados
	newInode.I_size = int32(size)
	newInode.I_type = [1]byte{1} // Tipo archivo
	newInode.I_block[0] = newBlockIndex

	// Escribe el nuevo inodo en el offset correcto para el nuevo inodo
	inodeOffset := int64(superblock.S_inode_start + newInodeIndex*int32(binary.Size(Structs.Inode{})))
	if err := Utilities.WriteObject(file, newInode, inodeOffset); err != nil {
		return fmt.Errorf("error al escribir el inodo del archivo: %v", err)
	}

	// Inicializa el nuevo fileblock
	var newFileblock Structs.Fileblock
	copy(newFileblock.B_content[:], content)

	// Escribe el nuevo fileblock en el archivo
	blockOffset := int64(superblock.S_block_start + newBlockIndex*int32(binary.Size(Structs.Fileblock{})))
	if err := Utilities.WriteObject(file, newFileblock, blockOffset); err != nil {
		return fmt.Errorf("error al escribir el bloque de archivo: %v", err)
	}

	// Marca el inodo y el bloque como usados en los bitmaps
	if err := markInodeAsUsed(newInodeIndex, superblock, file); err != nil {
		return fmt.Errorf("error al marcar el inodo como usado: %v", err)
	}
	if err := markBlockAsUsed(newBlockIndex, superblock, file); err != nil {
		return fmt.Errorf("error al marcar el bloque como usado: %v", err)
	}

	// Actualiza el folderblock del inodo padre
	if err := updateParentFolderblock(fileName, parentInode, newInodeIndex, file, superblock); err != nil {
		return fmt.Errorf("error al actualizar el folderblock del inodo padre: %v", err)
	}

	// Escribir de nuevo el superbloque con las modificaciones
	if err := Utilities.WriteObject(file, superblock, partitionStart); err != nil {
		return fmt.Errorf("error al actualizar el superbloque final: %v", err)
	}

	return nil
}

// Función auxiliar para marcar un inodo como usado
func markInodeAsUsed(inodeIndex int32, superblock Structs.Superblock, file *os.File) error {
	bitmapOffset := int64(superblock.S_bm_inode_start + inodeIndex)
	if err := Utilities.WriteObject(file, byte(1), bitmapOffset); err != nil {
		return fmt.Errorf("error al marcar el inodo en el bitmap: %v", err)
	}
	superblock.S_free_inodes_count-- // Disminuir el contador de inodos libres
	return nil
}

// Función auxiliar para marcar un bloque como usado
func markBlockAsUsed(blockIndex int32, superblock Structs.Superblock, file *os.File) error {
	bitmapOffset := int64(superblock.S_bm_block_start + blockIndex)
	if err := Utilities.WriteObject(file, byte(1), bitmapOffset); err != nil {
		return fmt.Errorf("error al marcar el bloque en el bitmap: %v", err)
	}
	superblock.S_free_blocks_count-- // Disminuir el contador de bloques libres
	return nil
}

func Cat(files []string) (string, error) {
	var logs string
	logs += "======INICIO CAT======\n"

	// Verificar si hay una partición logueada
	if User.CurrentLoggedPartitionID == "" {
		errMsg := "No hay ninguna partición logueada"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Buscar la partición montada por ID
	partition, err := getMountedPartition()
	if err != nil {
		return "", fmt.Errorf("error al obtener la partición montada: %w", err)
	}

	// Abrir archivo binario de la partición
	file, err := Utilities.OpenFile(partition.Path)
	if err != nil {
		errMsg := fmt.Sprintf("Error al abrir el archivo: %v", err)
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}
	defer file.Close()

	// Leer el MBR para obtener el inicio de la partición
	var mbr Structs.MBR
	if err := Utilities.ReadObject(file, &mbr, 0); err != nil {
		return "", fmt.Errorf("error al leer el MBR: %v", err)
	}

	// Encontrar la partición montada
	var partitionStart int64
	for _, part := range mbr.Partitions {
		if strings.TrimSpace(string(part.Id[:])) == partition.ID {
			partitionStart = int64(part.Start)
			break
		}
	}

	if partitionStart == 0 {
		return "", fmt.Errorf("no se encontró la partición montada")
	}

	// Leer el superbloque
	var superblock Structs.Superblock
	if err := Utilities.ReadObject(file, &superblock, partitionStart); err != nil {
		return "", fmt.Errorf("error al leer el superbloque: %v", err)
	}

	// Iterar sobre cada archivo proporcionado
	for _, filepath := range files {
		logs += fmt.Sprintf("Leyendo archivo: %s\n", filepath)

		// Buscar el archivo por su ruta y obtener el índice del inodo
		indexInode := User.InitSearch(filepath, file, superblock)
		if indexInode == -1 {
			errMsg := fmt.Sprintf("Error: No se encontró el archivo %s", filepath)
			logs += errMsg + "\n"
			continue
		}

		// Leer el inodo del archivo
		var inode Structs.Inode
		inodeOffset := int64(superblock.S_inode_start + indexInode*int32(binary.Size(Structs.Inode{})))
		if err := Utilities.ReadObject(file, &inode, inodeOffset); err != nil {
			errMsg := fmt.Sprintf("Error al leer el inodo del archivo %s", filepath)
			logs += errMsg + "\n"
			continue
		}

		// Verificar permisos de lectura del archivo (asumiendo que el permiso de lectura es el primer bit)
		if inode.I_perm[0] != 'r' {
			errMsg := fmt.Sprintf("Error: No tiene permiso de lectura para el archivo %s", filepath)
			logs += errMsg + "\n"
			continue
		}

		// Obtener y mostrar el contenido del archivo
		content := User.GetInodeFileData(inode, file, superblock)
		logs += fmt.Sprintf("Contenido del archivo %s:\n%s\n", filepath, content)
	}

	logs += "======FIN CAT======\n"
	return logs, nil
}
