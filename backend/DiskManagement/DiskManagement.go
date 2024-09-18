package DiskManagement

import (
	"backend/Structs"
	"backend/Utilities"
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

// Estructura para representar una partición montada
type MountedPartition struct {
	Path     string
	Name     string
	ID       string
	Status   byte // 0: no montada, 1: montada
	LoggedIn bool // true: usuario ha iniciado sesión, false: no ha iniciado sesión

}

// GetMountedPartitionByID busca una partición montada por su ID y la devuelve.
func GetMountedPartitionByID(id string) (*MountedPartition, error) {
	// Iterar sobre todas las particiones montadas
	for _, partitions := range mountedPartitions {
		for _, partition := range partitions {
			if partition.ID == id {
				// Retorna un puntero a la partición encontrada
				return &partition, nil
			}
		}
	}
	// Si no se encontró la partición, devolver un error
	return nil, fmt.Errorf("no se encontró ninguna partición montada con el ID %s", id)
}

// Mapa para almacenar las particiones montadas, organizadas por disco
var mountedPartitions = make(map[string][]MountedPartition)

// Función para imprimir las particiones montadas
// Función para obtener las particiones montadas como string
func GetMountedPartitionsString() string {
	var result strings.Builder

	result.WriteString("Particiones montadas:\n")
	if len(mountedPartitions) == 0 {
		result.WriteString("No hay particiones montadas.\n")
		return result.String()
	}

	for diskID, partitions := range mountedPartitions {
		result.WriteString(fmt.Sprintf("Disco ID: %s\n", diskID))
		for _, partition := range partitions {
			loginStatus := "No"
			if partition.LoggedIn {
				loginStatus = "Sí"
			}
			result.WriteString(fmt.Sprintf(" - Partición Name: %s, ID: %s, Path: %s, Status: %c, LoggedIn: %s\n",
				partition.Name, partition.ID, partition.Path, partition.Status, loginStatus))
		}
	}
	result.WriteString("\n")
	return result.String()
}

// Función para obtener las particiones montadas
func GetMountedPartitions() map[string][]MountedPartition {
	return mountedPartitions
}
func Rmdisk(path string) (string, error) {
	fmt.Println("======Start RMDISK======")
	fmt.Println("Path:", path)

	// Verificar si el archivo existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("Error: El DISCO no existe en la ruta especificada.")
		return "Error: El DISCO no existe en la ruta especificada.", nil
	}

	// Eliminar el archivo de disco
	err := os.Remove(path)
	if err != nil {
		fmt.Println("Error: No se pudo eliminar el archivo:", err)
		return "Error: No se pudo eliminar el archivo", err
	}
	fmt.Println("Disco eliminado exitosamente.")

	fmt.Println("======End RMDISK======")
	return "Disco eliminado exitosamente en: " + path, nil
}

func MarkPartitionAsLoggedOut(id string) error {
	for diskPath, partitions := range mountedPartitions {
		for i, partition := range partitions {
			if partition.ID == id {
				mountedPartitions[diskPath][i].LoggedIn = false
				return nil
			}
		}
	}
	return fmt.Errorf("no se encontró ninguna partición montada con el ID %s", id)
}

// Función para marcar una partición como logueada
func MarkPartitionAsLoggedIn(id string) {
	for diskID, partitions := range mountedPartitions {
		for i, partition := range partitions {
			if partition.ID == id {
				mountedPartitions[diskID][i].LoggedIn = true
				fmt.Printf("Partición con ID %s marcada como logueada.\n", id)
				return
			}
		}
	}
	fmt.Printf("No se encontró la partición con ID %s para marcarla como logueada.\n", id)
}

func Mkdisk(size int, fit string, unit string, path string) (string, error) {
	// Variable para acumular los mensajes
	var logs string

	// Agregar los mensajes de inicio al log
	logs += "======INICIO MKDISK======\n"
	logs += fmt.Sprintf("Size: %d\n", size)
	logs += fmt.Sprintf("Fit: %s\n", fit)
	logs += fmt.Sprintf("Unit: %s\n", unit)
	logs += fmt.Sprintf("Path: %s\n", path)

	// Validar fit bf/ff/wf
	if fit != "bf" && fit != "wf" && fit != "ff" {
		errMsg := "Error: Fit debe ser bf, wf o ff"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Validar size > 0
	if size <= 0 {
		errMsg := "Error: Size debe ser mayor a 0"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Validar unidad k - m
	if unit != "k" && unit != "m" {
		errMsg := "Error: Las unidades válidas son k o m"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Crear archivo
	err := Utilities.CreateFile(path)
	if err != nil {
		errMsg := fmt.Sprintf("Error al crear el archivo: %v", err)
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Asignar tamaño
	if unit == "k" {
		size = size * 1024
	} else {
		size = size * 1024 * 1024
	}

	// Abrir archivo binario
	file, err := Utilities.OpenFile(path)
	if err != nil {
		errMsg := fmt.Sprintf("Error al abrir el archivo: %v", err)
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}
	defer file.Close()

	// Escribir los 0 en el archivo
	for i := 0; i < size; i++ {
		if err := Utilities.WriteObject(file, byte(0), int64(i)); err != nil {
			errMsg := fmt.Sprintf("Error al escribir en el archivo: %v", err)
			logs += errMsg + "\n"
			return logs, fmt.Errorf(errMsg)
		}
	}

	// Crear MBR
	var newMRB Structs.MBR
	newMRB.MbrSize = int32(size)
	newMRB.Signature = rand.Int31() // Número random rand.Int31() genera solo números no negativos
	copy(newMRB.Fit[:], fit)

	// Obtener la fecha del sistema en formato YYYY-MM-DD
	currentTime := time.Now()
	formattedDate := currentTime.Format("2006-01-02")
	copy(newMRB.CreationDate[:], formattedDate)

	// Escribir el MBR en el archivo
	if err := Utilities.WriteObject(file, newMRB, 0); err != nil {
		errMsg := fmt.Sprintf("Error al escribir el MBR en el archivo: %v", err)
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	var TempMBR Structs.MBR
	// Leer el MBR del archivo
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		errMsg := fmt.Sprintf("Error al leer el MBR del archivo: %v", err)
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Formatear los datos del MBR manualmente y agregar al log
	logs += fmt.Sprintf("MBR Size: %d\n", TempMBR.MbrSize)
	logs += fmt.Sprintf("MBR Signature: %d\n", TempMBR.Signature)
	logs += fmt.Sprintf("MBR Fit: %s\n", string(TempMBR.Fit[:]))
	logs += fmt.Sprintf("MBR Creation Date: %s\n", string(TempMBR.CreationDate[:]))

	logs += "======FIN MKDISK======\n"
	return logs + fmt.Sprintf("MKDISK: Disco creado exitosamente en: %s", path), nil
}

func Fdisk(size int, path string, name string, unit string, type_ string, fit string) (string, error) {
	// Variable para acumular los mensajes
	var logs string

	// Agregar los mensajes de inicio al log
	logs += "======Start FDISK======\n"
	logs += fmt.Sprintf("Size: %d\n", size)
	logs += fmt.Sprintf("Path: %s\n", path)
	logs += fmt.Sprintf("Name: %s\n", name)
	logs += fmt.Sprintf("Unit: %s\n", unit)
	logs += fmt.Sprintf("Type: %s\n", type_)
	logs += fmt.Sprintf("Fit: %s\n", fit)

	// Validar fit (b/w/f)
	if fit != "b" && fit != "f" && fit != "w" {
		errMsg := "Error: Fit must be 'b', 'f', or 'w'"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Validar size > 0
	if size <= 0 {
		errMsg := "Error: Size must be greater than 0"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Validar unit (b/k/m)
	if unit != "b" && unit != "k" && unit != "m" {
		errMsg := "Error: Unit must be 'b', 'k', or 'm'"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Ajustar el tamaño en bytes
	if unit == "k" {
		size = size * 1024
	} else if unit == "m" {
		size = size * 1024 * 1024
	}

	// Abrir el archivo binario en la ruta proporcionada
	file, err := Utilities.OpenFile(path)
	if err != nil {
		errMsg := fmt.Sprintf("Error: Could not open file at path: %s", path)
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}
	defer file.Close()

	var TempMBR Structs.MBR
	// Leer el objeto desde el archivo binario
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		errMsg := "Error: Could not read MBR from file"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Formatear el MBR y agregarlo al log
	logs += fmt.Sprintf("MBR Size: %d\n", TempMBR.MbrSize)
	logs += fmt.Sprintf("MBR Signature: %d\n", TempMBR.Signature)
	logs += fmt.Sprintf("MBR Fit: %s\n", string(TempMBR.Fit[:]))
	logs += fmt.Sprintf("MBR Creation Date: %s\n", string(TempMBR.CreationDate[:]))
	logs += "-------------\n"

	// Validaciones de las particiones
	var primaryCount, extendedCount, totalPartitions int
	var usedSpace int32 = 0

	for i := 0; i < 4; i++ {
		if TempMBR.Partitions[i].Size != 0 {
			totalPartitions++
			usedSpace += TempMBR.Partitions[i].Size

			if TempMBR.Partitions[i].Type[0] == 'p' {
				primaryCount++
			} else if TempMBR.Partitions[i].Type[0] == 'e' {
				extendedCount++
			}
		}
	}

	// Validar que no se exceda el número máximo de particiones primarias y extendidas
	if totalPartitions >= 4 {
		errMsg := "Error: No se pueden crear más de 4 particiones primarias o extendidas en total."
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Validar que solo haya una partición extendida
	if type_ == "e" && extendedCount > 0 {
		errMsg := "Error: Solo se permite una partición extendida por disco."
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Validar que no se pueda crear una partición lógica sin una extendida
	if type_ == "l" && extendedCount == 0 {
		errMsg := "Error: No se puede crear una partición lógica sin una partición extendida."
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Validar que el tamaño de la nueva partición no exceda el tamaño del disco
	if usedSpace+int32(size) > TempMBR.MbrSize {
		errMsg := "Error: No hay suficiente espacio en el disco para crear esta partición."
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Determinar la posición de inicio de la nueva partición
	var gap int32 = int32(binary.Size(TempMBR))
	if totalPartitions > 0 {
		gap = TempMBR.Partitions[totalPartitions-1].Start + TempMBR.Partitions[totalPartitions-1].Size
	}

	// Encontrar una posición vacía para la nueva partición
	for i := 0; i < 4; i++ {
		if TempMBR.Partitions[i].Size == 0 {
			if type_ == "p" || type_ == "e" {
				// Crear partición primaria o extendida
				TempMBR.Partitions[i].Size = int32(size)
				TempMBR.Partitions[i].Start = gap
				copy(TempMBR.Partitions[i].Name[:], name)
				copy(TempMBR.Partitions[i].Fit[:], fit)
				copy(TempMBR.Partitions[i].Status[:], "0")
				copy(TempMBR.Partitions[i].Type[:], type_)
				TempMBR.Partitions[i].Correlative = int32(totalPartitions + 1)

				if type_ == "e" {
					// Inicializar el primer EBR en la partición extendida
					ebrStart := gap // El primer EBR se coloca al inicio de la partición extendida
					ebr := Structs.EBR{
						PartFit:   fit[0],
						PartStart: ebrStart,
						PartSize:  0,
						PartNext:  -1,
					}
					copy(ebr.PartName[:], "")
					Utilities.WriteObject(file, ebr, int64(ebrStart))
				}

				break
			}
		}
	}

	// Manejar la creación de particiones lógicas dentro de una partición extendida
	if type_ == "l" {
		for i := 0; i < 4; i++ {
			if TempMBR.Partitions[i].Type[0] == 'e' {
				ebrPos := TempMBR.Partitions[i].Start
				var ebr Structs.EBR
				for {
					Utilities.ReadObject(file, &ebr, int64(ebrPos))
					if ebr.PartNext == -1 {
						break
					}
					ebrPos = ebr.PartNext
				}

				// Calcular la posición de inicio de la nueva partición lógica
				newEBRPos := ebr.PartStart + ebr.PartSize                    // El nuevo EBR se coloca después de la partición lógica anterior
				logicalPartitionStart := newEBRPos + int32(binary.Size(ebr)) // El inicio de la partición lógica es justo después del EBR

				// Ajustar el siguiente EBR
				ebr.PartNext = newEBRPos
				Utilities.WriteObject(file, ebr, int64(ebrPos))

				// Crear y escribir el nuevo EBR
				newEBR := Structs.EBR{
					PartFit:   fit[0],
					PartStart: logicalPartitionStart,
					PartSize:  int32(size),
					PartNext:  -1,
				}
				copy(newEBR.PartName[:], name)
				Utilities.WriteObject(file, newEBR, int64(newEBRPos))

				// Agregar el nuevo EBR creado al log
				logs += "Nuevo EBR creado:\n"
				logs += fmt.Sprintf("EBR Start: %d\n", newEBR.PartStart)
				logs += fmt.Sprintf("EBR Size: %d\n", newEBR.PartSize)
				logs += fmt.Sprintf("EBR Next: %d\n", newEBR.PartNext)
				logs += "\n"

				// Imprimir todos los EBRs en la partición extendida
				logs += "Imprimiendo todos los EBRs en la partición extendida:\n"
				ebrPos = TempMBR.Partitions[i].Start
				for {
					err := Utilities.ReadObject(file, &ebr, int64(ebrPos))
					if err != nil {
						logs += fmt.Sprintf("Error al leer EBR: %v\n", err)
						break
					}
					logs += fmt.Sprintf("EBR Start: %d, Size: %d, Next: %d\n", ebr.PartStart, ebr.PartSize, ebr.PartNext)
					if ebr.PartNext == -1 {
						break
					}
					ebrPos = ebr.PartNext
				}

				break
			}
		}
		logs += "\n"
	}

	// Sobrescribir el MBR
	if err := Utilities.WriteObject(file, TempMBR, 0); err != nil {
		errMsg := "Error: Could not write MBR to file"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	var TempMBR2 Structs.MBR
	// Leer el objeto nuevamente para verificar
	if err := Utilities.ReadObject(file, &TempMBR2, 0); err != nil {
		errMsg := "Error: Could not read MBR from file after writing"
		logs += errMsg + "\n"
		return logs, fmt.Errorf(errMsg)
	}

	// Agregar el MBR actualizado al log
	logs += fmt.Sprintf("MBR Size: %d\n", TempMBR2.MbrSize)
	logs += fmt.Sprintf("MBR Signature: %d\n", TempMBR2.Signature)
	logs += fmt.Sprintf("MBR Fit: %s\n", string(TempMBR2.Fit[:]))
	logs += fmt.Sprintf("MBR Creation Date: %s\n", string(TempMBR2.CreationDate[:]))

	logs += "======FIN FDISK======\n"
	return logs + fmt.Sprintf("FDISK: Partición %s creada exitosamente en: %s", name, path), nil
}

func Mount(path string, name string) (string, error) {
	file, err := Utilities.OpenFile(path)
	if err != nil {
		mountedPartitionsStr := GetMountedPartitionsString()
		return "Error: No se pudo abrir el archivo en la ruta:\n" + mountedPartitionsStr, nil
	}
	defer file.Close()

	var TempMBR Structs.MBR
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		mountedPartitionsStr := GetMountedPartitionsString()
		return "Error: No se pudo leer el MBR desde el archivo:\n" + mountedPartitionsStr, nil
	}

	fmt.Printf("Buscando partición con nombre: '%s'\n", name)

	partitionFound := false
	var partition Structs.Partition
	var partitionIndex int

	// Convertir el nombre a comparar a un arreglo de bytes de longitud fija
	nameBytes := [16]byte{}
	copy(nameBytes[:], []byte(name))

	for i := 0; i < 4; i++ {
		if TempMBR.Partitions[i].Type[0] == 'p' && bytes.Equal(TempMBR.Partitions[i].Name[:], nameBytes[:]) {
			partition = TempMBR.Partitions[i]
			partitionIndex = i
			partitionFound = true
			break
		}
	}

	if !partitionFound {
		mountedPartitionsStr := GetMountedPartitionsString()
		return "Error: Partición no encontrada o no es una partición primaria:\n" + mountedPartitionsStr, nil
	}

	// Verificar si la partición ya está montada
	if partition.Status[0] == '1' {
		mountedPartitionsStr := GetMountedPartitionsString()
		return "Error: La partición ya está montada:\n" + mountedPartitionsStr, nil
	}

	// Generar el ID de la partición
	diskID := generateDiskID(path)

	// Verificar si ya se ha montado alguna partición de este disco
	mountedPartitionsInDisk := mountedPartitions[diskID]
	var letter byte

	if len(mountedPartitionsInDisk) == 0 {
		// Es un nuevo disco, asignar la siguiente letra disponible
		if len(mountedPartitions) == 0 {
			letter = 'a'
		} else {
			lastDiskID := getLastDiskID()
			lastLetter := mountedPartitions[lastDiskID][0].ID[len(mountedPartitions[lastDiskID][0].ID)-1]
			letter = lastLetter + 1
		}
	} else {
		// Utilizar la misma letra que las otras particiones montadas en el mismo disco
		letter = mountedPartitionsInDisk[0].ID[len(mountedPartitionsInDisk[0].ID)-1]
	}

	// Incrementar el número para esta partición
	carnet := "202212209"
	lastTwoDigits := carnet[len(carnet)-2:]
	partitionID := fmt.Sprintf("%s%d%c", lastTwoDigits, partitionIndex+1, letter)

	// Actualizar el estado de la partición a montada y asignar el ID
	partition.Status[0] = '1'
	copy(partition.Id[:], partitionID)
	TempMBR.Partitions[partitionIndex] = partition
	mountedPartitions[diskID] = append(mountedPartitions[diskID], MountedPartition{
		Path:   path,
		Name:   name,
		ID:     partitionID,
		Status: '1',
	})

	// Escribir el MBR actualizado al archivo
	if err := Utilities.WriteObject(file, TempMBR, 0); err != nil {
		mountedPartitionsStr := GetMountedPartitionsString()
		return "Error: No se pudo sobrescribir el MBR en el archivo:\n" + mountedPartitionsStr, nil
	}

	mountedPartitionsStr := GetMountedPartitionsString()
	return fmt.Sprintf("Partición montada con ID: %s\n%s", partitionID, mountedPartitionsStr), nil
}

// Función para obtener el ID del último disco montado
func getLastDiskID() string {
	var lastDiskID string
	for diskID := range mountedPartitions {
		lastDiskID = diskID
	}
	return lastDiskID
}

func generateDiskID(path string) string {
	return strings.ToLower(path)
}

func GetPartitionByID(id string) *MountedPartition {
	for _, partitions := range mountedPartitions {
		for _, partition := range partitions {
			if partition.ID == id {
				return &partition
			}
		}
	}
	return nil
}

// Funciones para generar los reportes (cada función llamará a Graphviz)
// generateMBRReport genera un reporte del MBR y lo guarda en la ruta especificada
func GenerateMBRReport(path string, partition MountedPartition) error {
	// Crear las carpetas padre si no existen
	err := createDirectoryIfNotExists(path)
	if err != nil {
		return err
	}

	// Obtener el nombre base del archivo sin la extensión y la imagen de salida
	dotFileName, outputImage := getFileNames(path)

	// Leer el MBR desde el archivo binario correspondiente
	file, err := os.Open(partition.Path)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo del disco: %v", err)
	}
	defer file.Close()

	var mbr Structs.MBR
	if err := binary.Read(file, binary.LittleEndian, &mbr); err != nil {
		return fmt.Errorf("error al leer el MBR desde el archivo: %v", err)
	}

	// Crear el contenido DOT con una tabla
	dotContent := fmt.Sprintf(`digraph G {
        node [shape=plaintext]
        tabla [label=<
            <table border="0" cellborder="1" cellspacing="0">
                <tr><td colspan="2" bgcolor="lightblue"> REPORTE MBR </td></tr>
                <tr><td bgcolor="lightgrey">mbr_tamano</td><td>%d</td></tr>
                <tr><td bgcolor="lightgrey">mrb_fecha_creacion</td><td>%s</td></tr>
                <tr><td bgcolor="lightgrey">mbr_disk_signature</td><td>%d</td></tr>
            `, mbr.MbrSize, string(mbr.CreationDate[:]), mbr.Signature)

	// Iterar sobre todas las particiones y mostrar sus datos, incluso si no están definidas
	for i, part := range mbr.Partitions {
		// Convertir los valores a caracteres o mostrar un valor predeterminado si están vacíos
		partStatus := '0'
		if part.Status[0] != 0 {
			partStatus = rune(part.Status[0])
		}

		partType := '-'
		if part.Type[0] != 0 {
			partType = rune(part.Type[0])
		}

		partFit := '-'
		if part.Fit[0] != 0 {
			partFit = rune(part.Fit[0])
		}

		partName := strings.TrimRight(string(part.Name[:]), "\x00")

		// Definir el color de fondo según el tipo de partición
		bgColor := "white"
		if partType == 'e' {
			bgColor = "lightgreen"
		} else if partType == 'p' {
			bgColor = "lightyellow"
		}

		// Agregar la partición a la tabla, mostrando valores por defecto si es necesario
		dotContent += fmt.Sprintf(`
                <tr><td colspan="2" bgcolor="%s"> PARTICIÓN %d </td></tr>
                <tr><td bgcolor="lightgrey">part_status</td><td>%c</td></tr>
                <tr><td bgcolor="lightgrey">part_type</td><td>%c</td></tr>
                <tr><td bgcolor="lightgrey">part_fit</td><td>%c</td></tr>
                <tr><td bgcolor="lightgrey">part_start</td><td>%d</td></tr>
                <tr><td bgcolor="lightgrey">part_size</td><td>%d</td></tr>
                <tr><td bgcolor="lightgrey">part_name</td><td>%s</td></tr>
            `, bgColor, i+1, partStatus, partType, partFit, part.Start, part.Size, partName)

		// Si la partición es extendida, buscar EBRs y mostrar particiones lógicas
		if partType == 'e' {
			err = showLogicalPartitions(file, part.Start, &dotContent)
			if err != nil {
				return fmt.Errorf("error al mostrar particiones lógicas: %v", err)
			}
		}
	}

	// Cerrar la tabla y el contenido DOT
	dotContent += "</table>>] }"

	// Guardar el contenido DOT en un archivo en la carpeta especificada
	dotFilePath := filepath.Join(path, dotFileName)
	fileDot, err := os.Create(dotFilePath)
	if err != nil {
		return fmt.Errorf("error al crear el archivo DOT: %v", err)
	}
	defer fileDot.Close()

	_, err = fileDot.WriteString(dotContent)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo DOT: %v", err)
	}

	// Ejecutar el comando Graphviz para generar la imagen en la misma carpeta
	outputImagePath := filepath.Join(path, outputImage)
	cmd := exec.Command("dot", "-Tpng", dotFilePath, "-o", outputImagePath)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Imagen del reporte MBR generada en:", outputImagePath)
	return nil
}

// showLogicalPartitions muestra las particiones lógicas dentro de una partición extendida
func showLogicalPartitions(file *os.File, extendedStart int32, dotContent *string) error {
	var ebr Structs.EBR
	ebrPosition := extendedStart

	for {
		// Leer el EBR en la posición actual
		err := readEBR(file, &ebr, ebrPosition)
		if err != nil {
			return fmt.Errorf("error al leer EBR: %v", err)
		}

		// Mostrar la partición lógica solo si tiene un tamaño mayor a cero
		if ebr.PartSize > 0 {
			ebrName := strings.TrimRight(string(ebr.PartName[:]), "\x00")
			*dotContent += fmt.Sprintf(`
                <tr><td colspan="2" bgcolor="lightcoral"> PARTICIÓN LÓGICA </td></tr>
                <tr><td bgcolor="lightgrey">part_fit</td><td>%c</td></tr>
                <tr><td bgcolor="lightgrey">part_start</td><td>%d</td></tr>
                <tr><td bgcolor="lightgrey">part_size</td><td>%d</td></tr>
                <tr><td bgcolor="lightgrey">part_next</td><td>%d</td></tr>
                <tr><td bgcolor="lightgrey">part_name</td><td>%s</td></tr>
            `, ebr.PartFit, ebr.PartStart, ebr.PartSize, ebr.PartNext, ebrName)
		}

		// Si no hay más particiones lógicas, detener
		if ebr.PartNext == -1 {
			break
		}

		// Ir a la siguiente partición lógica
		ebrPosition = ebr.PartNext
	}

	return nil
}

// Función para leer un EBR desde una posición específica en el archivo
func readEBR(file *os.File, ebr *Structs.EBR, position int32) error {
	// Crear un buffer para leer los datos del EBR
	buffer := make([]byte, binary.Size(*ebr))

	// Leer los bytes del EBR desde la posición especificada
	_, err := file.ReadAt(buffer, int64(position))
	if err != nil {
		return err
	}

	// Decodificar los datos del buffer al EBR utilizando LittleEndian
	err = binary.Read(strings.NewReader(string(buffer)), binary.LittleEndian, ebr)
	if err != nil {
		return err
	}

	return nil
}

func createDirectoryIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}
	return nil
}

// Función para obtener los nombres del archivo DOT y la imagen de salida
func getFileNames(path string) (string, string) {
	baseName := filepath.Base(path)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	dotFileName := baseName + ".dot"
	outputImage := baseName + ".png"
	return dotFileName, outputImage
}

// GenerateDiskReport genera un reporte de la estructura de particiones del disco y lo guarda en la ruta especificada
func GenerateDiskReport(path string, partition *MountedPartition) error {
	// Crear las carpetas padre si no existen
	err := createDirectoryIfNotExists(path)
	if err != nil {
		return err
	}

	// Obtener el nombre base del archivo sin la extensión y la imagen de salida
	dotFileName, outputImage := getFileNames(path)

	// Leer el MBR desde el archivo binario correspondiente
	file, err := os.Open(partition.Path)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo del disco: %v", err)
	}
	defer file.Close()

	var mbr Structs.MBR
	if err := binary.Read(file, binary.LittleEndian, &mbr); err != nil {
		return fmt.Errorf("error al leer el MBR desde el archivo: %v", err)
	}

	// Crear el contenido DOT inicial con la estructura requerida
	dotContent := `digraph G {
		node [shape=plaintext];
		
		subgraph cluster_0 {
			label="Disco1.dsk";
			fontsize=20;
			
			tabla [label=<
				<TABLE BORDER="1" CELLBORDER="1" CELLSPACING="0" COLOR="blue">
					<TR>`

	// Variables para calcular el espacio total y usado
	var usedSpace int32
	var extendedPartition *Structs.Partition
	const mbrSize int32 = 159 // Tamaño fijo de la estructura MRB

	// Añadir MBR
	dotContent += fmt.Sprintf(`<TD BGCOLOR="lightblue">MBR</TD>`)
	usedSpace += mbrSize
	// Iterar sobre todas las particiones
	for i, part := range mbr.Partitions {
		if part.Size > 0 {
			usedSpace += part.Size
			percentage := float64(part.Size) / float64(mbr.MbrSize) * 100
			partType := rune(part.Type[0])
			partName := strings.TrimRight(string(part.Name[:]), "\x00")

			if partType == 'e' {
				extendedPartition = &mbr.Partitions[i]
				dotContent += fmt.Sprintf(`<TD><TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0"><TR><TD COLSPAN="5" BGCOLOR="lightgreen">Extendida %.2f%%</TD></TR><TR>`, percentage)
				err = addLogicalPartitions(file, extendedPartition.Start, extendedPartition.Size, &dotContent)
				if err != nil {
					return fmt.Errorf("error al mostrar particiones lógicas: %v", err)
				}
				dotContent += `</TR></TABLE></TD>`
			} else {
				dotContent += fmt.Sprintf(`<TD BGCOLOR="lightyellow">%s<BR/>%.2f%%</TD>`, partName, percentage)
			}
		}
	}

	// Calcular y mostrar el espacio libre al final
	freeSpace := mbr.MbrSize - usedSpace
	if freeSpace > 0 {
		freePercentage := float64(freeSpace) / float64(mbr.MbrSize) * 100
		dotContent += fmt.Sprintf(`<TD BGCOLOR="lightgray">Libre<BR/>%.2f%%</TD>`, freePercentage)
	}

	// Cerrar la tabla y el contenido DOT
	dotContent += `
					</TR>
				</TABLE>
			>];
		}
	}` // Guardar el contenido DOT en un archivo en la carpeta especificada
	dotFilePath := filepath.Join(path, dotFileName)
	fileDot, err := os.Create(dotFilePath)
	if err != nil {
		return fmt.Errorf("error al crear el archivo DOT: %v", err)
	}
	defer fileDot.Close()

	_, err = fileDot.WriteString(dotContent)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo DOT: %v", err)
	}

	// Ejecutar el comando Graphviz para generar la imagen en la misma carpeta
	outputImagePath := filepath.Join(path, outputImage)
	cmd := exec.Command("dot", "-Tpng", dotFilePath, "-o", outputImagePath)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Imagen del reporte de disco generada en:", outputImagePath)
	return nil
}

func addLogicalPartitions(file *os.File, extendedStart int32, extendedSize int32, dotContent *string) error {
	var ebr Structs.EBR
	ebrPosition := extendedStart
	remainingSize := extendedSize

	for {
		// Leer el EBR en la posición actual
		err := readEBR(file, &ebr, ebrPosition)
		if err != nil {
			return fmt.Errorf("error al leer EBR: %v", err)
		}

		// Mostrar EBR
		*dotContent += `<TD BGCOLOR="lightblue">EBR</TD>`

		// Mostrar la partición lógica si tiene un tamaño mayor a cero
		if ebr.PartSize > 0 {
			ebrName := strings.TrimRight(string(ebr.PartName[:]), "\x00")
			percentage := float64(ebr.PartSize) / float64(extendedSize) * 100
			*dotContent += fmt.Sprintf(`<TD BGCOLOR="lightcoral">%s<BR/>%.2f%%</TD>`, ebrName, percentage)
			remainingSize -= ebr.PartSize
		}

		// Si no hay más particiones lógicas, mostrar el espacio libre restante y detener
		if ebr.PartNext == -1 {
			if remainingSize > 0 {
				freePercentage := float64(remainingSize) / float64(extendedSize) * 100
				*dotContent += fmt.Sprintf(`<TD BGCOLOR="lightgray">Libre<BR/>%.2f%%</TD>`, freePercentage)
			}
			break
		}

		// Calcular y mostrar el espacio libre entre particiones lógicas
		freeSpace := ebr.PartNext - (ebrPosition + int32(unsafe.Sizeof(ebr)) + ebr.PartSize)
		if freeSpace > 0 {
			freePercentage := float64(freeSpace) / float64(extendedSize) * 100
			*dotContent += fmt.Sprintf(`<TD BGCOLOR="lightgray">Libre<BR/>%.2f%%</TD>`, freePercentage)
		}

		// Ir a la siguiente partición lógica
		ebrPosition = ebr.PartNext
	}

	return nil
}

// GenerateInodeReport genera un reporte visual de los inodos y lo guarda en la ruta especificada
func GenerateInodeReport(path string, partition *MountedPartition) error {
	// Crear las carpetas padre si no existen
	if err := createDirectoryIfNotExists(path); err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}

	// Obtener el nombre base del archivo sin la extensión y la imagen de salida
	dotFileName, outputImage := getFileNames(path)

	// Abrir el archivo binario del disco desde la partición montada
	file, err := os.Open(partition.Path)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo del disco: %v", err)
	}
	defer file.Close()

	// Leer el Superblock para obtener la información de los inodos
	var superblock Structs.Superblock
	superblockOffset := int64(binary.Size(Structs.MBR{})) // Ajusta según la posición real del Superblock
	if err := Utilities.ReadObject(file, &superblock, superblockOffset); err != nil {
		return fmt.Errorf("error al leer el Superblock: %v", err)
	}

	// Iniciar el contenido DOT con configuraciones de color
	dotContent := `digraph G {
		node [shape=plaintext];
		rankdir=LR; // Layout de izquierda a derecha
	`

	// Iterar sobre cada inodo y generar su representación en Graphviz
	for i := int32(0); i < superblock.S_inodes_count; i++ {
		var inode Structs.Inode
		inodeOffset := superblock.S_inode_start + i*superblock.S_inode_size
		if err := Utilities.ReadObject(file, &inode, int64(inodeOffset)); err != nil {
			return fmt.Errorf("error al leer inodo %d: %v", i, err)
		}

		// Verificar si el inodo está vacío (sin uso)
		if isEmptyInode(inode) {
			continue // Omitir inodos vacíos
		}

		// Agregar representación del inodo al contenido DOT
		dotContent += formatInodeToDot(i, inode)

		// Conectar con el siguiente inodo si no es el último
		if i < superblock.S_inodes_count-1 {
			dotContent += fmt.Sprintf("inode%d -> inode%d;\n", i, i+1)
		}
	}

	// Cerrar el contenido DOT
	dotContent += "}"

	// Crear el archivo DOT
	dotFilePath := filepath.Join(path, dotFileName)
	if err := os.WriteFile(dotFilePath, []byte(dotContent), 0644); err != nil {
		return fmt.Errorf("error al crear o escribir en el archivo DOT: %v", err)
	}

	// Ejecutar el comando Graphviz para generar la imagen en la misma carpeta
	outputImagePath := filepath.Join(path, outputImage)
	if err := exec.Command("dot", "-Tpng", dotFilePath, "-o", outputImagePath).Run(); err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Imagen del reporte de inodos generada en:", outputImagePath)
	return nil
}

// formatInodeToDot genera la representación en formato DOT de un inodo dado
func formatInodeToDot(index int32, inode Structs.Inode) string {
	// Convertir tiempos a string
	atime := cleanDateString(string(inode.I_atime[:]))
	ctime := cleanDateString(string(inode.I_ctime[:]))
	mtime := cleanDateString(string(inode.I_mtime[:]))

	// Definir el contenido DOT para el inodo actual con colores
	var typeStr string
	switch inode.I_type[0] {
	case '\x00':
		typeStr = "Directorio"
	case '\x01':
		typeStr = "Archivo"
	default:
		typeStr = "Desconocido" // Para cualquier otro valor inesperado
	}

	// Definir el contenido DOT para el inodo actual con el tipo corregido
	content := fmt.Sprintf(`inode%d [label=<
		<table border="0" cellborder="1" cellspacing="0">
			<tr><td colspan="2" bgcolor="#B0C4DE"><b>REPORTE INODO %d</b></td></tr>
			<tr><td bgcolor="#F5F5F5"><b>UID</b></td><td bgcolor="#FFFFFF">%d</td></tr>
			<tr><td bgcolor="#F5F5F5"><b>GID</b></td><td bgcolor="#FFFFFF">%d</td></tr>
			<tr><td bgcolor="#F5F5F5"><b>Size</b></td><td bgcolor="#FFFFFF">%d</td></tr>
			<tr><td bgcolor="#F5F5F5"><b>Atime</b></td><td bgcolor="#FFFFFF">%s</td></tr>
			<tr><td bgcolor="#F5F5F5"><b>Ctime</b></td><td bgcolor="#FFFFFF">%s</td></tr>
			<tr><td bgcolor="#F5F5F5"><b>Mtime</b></td><td bgcolor="#FFFFFF">%s</td></tr>
			<tr><td bgcolor="#F5F5F5"><b>Type</b></td><td bgcolor="#FFFFFF">%s</td></tr>
			<tr><td bgcolor="#F5F5F5"><b>Perm</b></td><td bgcolor="#FFFFFF">%s</td></tr>
			<tr><td colspan="2" bgcolor="#D3D3D3"><b>BLOQUES DIRECTOS</b></td></tr>
		`, index, index, inode.I_uid, inode.I_gid, inode.I_size, atime, ctime, mtime, typeStr, string(inode.I_perm[:]))

	// Agregar los bloques directos a la tabla
	for j, block := range inode.I_block[:12] {
		color := "#FFFFFF" // Color de fondo para bloques asignados
		if block == -1 {
			color = "#FFCCCB" // Resaltar bloques no asignados en rojo claro
		}
		content += fmt.Sprintf("<tr><td bgcolor=\"#F5F5F5\">Bloque %d</td><td bgcolor=\"%s\">%d</td></tr>", j+1, color, block)
	}

	// Agregar bloques indirectos, doble y triple con colores distintos
	content += fmt.Sprintf(`
		<tr><td colspan="2" bgcolor="#D3D3D3"><b>BLOQUE INDIRECTO</b></td></tr>
		<tr><td bgcolor="#F5F5F5">13</td><td bgcolor="#FFFFFF">%d</td></tr>
		<tr><td colspan="2" bgcolor="#D3D3D3"><b>BLOQUE INDIRECTO DOBLE</b></td></tr>
		<tr><td bgcolor="#F5F5F5">14</td><td bgcolor="#FFFFFF">%d</td></tr>
		<tr><td colspan="2" bgcolor="#D3D3D3"><b>BLOQUE INDIRECTO TRIPLE</b></td></tr>
		<tr><td bgcolor="#F5F5F5">15</td><td bgcolor="#FFFFFF">%d</td></tr>
		</table>>];
	`, inode.I_block[12], inode.I_block[13], inode.I_block[14])

	return content
}

// isEmptyInode verifica si un inodo está vacío o no contiene información útil
func isEmptyInode(inode Structs.Inode) bool {
	// Comprobar si el inodo está vacío basado en los valores típicos que indican inactividad
	if inode.I_uid == 0 && inode.I_gid == 0 && inode.I_size == 0 {
		// Comprobar si todos los bloques son -1 (no asignados)
		for _, block := range inode.I_block {
			if block != -1 {
				return false // No está vacío, tiene al menos un bloque asignado
			}
		}
		return true // Todos los bloques son -1 y los valores principales son cero
	}
	return false // El inodo tiene información relevante
}

// cleanDateString elimina caracteres no deseados de las cadenas de fechas
func cleanDateString(date string) string {
	// Remueve caracteres nulos y espacios en blanco innecesarios
	return strings.TrimRight(date, "\x00 ")
}

// cleanString elimina caracteres no deseados de cualquier cadena
func cleanString(s string) string {
	// Remueve caracteres nulos y espacios en blanco innecesarios
	return strings.TrimRight(s, "\x00 ")
}

// GenerateBlockReport genera un reporte visual de los bloques y lo guarda en la ruta especificada
func GenerateBlockReport(path string, partition *MountedPartition) error {
	// Crear las carpetas padre si no existen
	if err := createDirectoryIfNotExists(path); err != nil {
		return fmt.Errorf("Error al crear directorios: %v", err)
	}

	// Obtener el nombre base del archivo sin la extensión y la imagen de salida
	dotFileName, outputImage := getFileNames(path)

	// Abrir el archivo binario del disco desde la partición montada
	file, err := os.Open(partition.Path)
	if err != nil {
		return fmt.Errorf("Error al abrir el archivo del disco: %v", err)
	}
	defer file.Close()

	// Leer el Superblock para obtener la información de los bloques
	var superblock Structs.Superblock
	superblockOffset := int64(binary.Size(Structs.MBR{})) // Ajusta según la posición real del Superblock
	if err := Utilities.ReadObject(file, &superblock, superblockOffset); err != nil {
		return fmt.Errorf("Error al leer el Superblock: %v", err)
	}

	// Iniciar el contenido DOT con configuraciones de color y orden horizontal
	dotContent := `digraph G {
		rankdir=LR; // Layout de izquierda a derecha
		node [shape=plaintext];
	`

	// Variable para almacenar el nombre del nodo anterior para conectar los bloques
	var previousBlock string

	// Iterar sobre cada bloque y generar su representación en Graphviz
	for i := int32(0); i < superblock.S_blocks_count; i++ {
		var fileBlock Structs.Fileblock
		blockOffset := superblock.S_block_start + i*superblock.S_block_size
		if err := Utilities.ReadObject(file, &fileBlock, int64(blockOffset)); err != nil {
			return fmt.Errorf("Error al leer bloque %d: %v", i, err)
		}

		// Limpiar el contenido del bloque para eliminar caracteres no imprimibles
		blockContent := cleanStringb(string(fileBlock.B_content[:]))
		if blockContent != "" {
			// Agregar representación del bloque al contenido DOT
			blockName := fmt.Sprintf("block%d", i)
			dotContent += formatBlockToDot(i, blockContent)

			// Conectar el bloque anterior con el actual
			if previousBlock != "" {
				dotContent += fmt.Sprintf("%s -> %s;\n", previousBlock, blockName)
			}

			// Actualizar el nombre del nodo anterior
			previousBlock = blockName
		}
	}

	// Cerrar el contenido DOT
	dotContent += "}"

	// Crear el archivo DOT
	dotFilePath := filepath.Join(path, dotFileName)
	if err := os.WriteFile(dotFilePath, []byte(dotContent), 0644); err != nil {
		return fmt.Errorf("Error al crear o escribir en el archivo DOT: %v", err)
	}

	// Ejecutar el comando Graphviz para generar la imagen en la misma carpeta
	outputImagePath := filepath.Join(path, outputImage)
	if err := exec.Command("dot", "-Tpng", dotFilePath, "-o", outputImagePath).Run(); err != nil {
		return fmt.Errorf("Error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Imagen del reporte de bloques generada en:", outputImagePath)
	return nil
}

// formatBlockToDot genera la representación en formato DOT de un bloque dado
func formatBlockToDot(index int32, content string) string {
	// Definir el contenido DOT para el bloque actual con colores y separarlo horizontalmente
	blockDot := fmt.Sprintf(`block%d [label=<
		<table border="1" cellborder="1" cellspacing="0">
			<tr><td colspan="2" bgcolor="#B0C4DE"><b>REPORTE BLOQUE %d</b></td></tr>
			<tr><td bgcolor="#F5F5F5"><b>Contenido</b></td><td bgcolor="#FFFFFF">%s</td></tr>
		</table>>];
	`, index, index, content)

	return blockDot
}

// cleanString elimina o reemplaza caracteres no imprimibles de una cadena
func cleanStringb(s string) string {
	// Reemplaza o elimina caracteres no imprimibles
	cleaned := strings.Map(func(r rune) rune {
		if r < 32 || r > 126 { // Rango de caracteres imprimibles ASCII estándar
			return -1 // -1 indica que el carácter debe eliminarse
		}
		return r
	}, s)
	return cleaned
}

func GenerateBMInodeReport(path string, partition MountedPartition) {
	// Implementación pendiente
}

func GenerateBMBlockReport(path string, partition MountedPartition) {
	// Implementación pendiente
}

// GenerateSuperblockReport genera un reporte del Superbloque y lo guarda en la ruta especificada
func GenerateSuperblockReport(path string, partition *MountedPartition) error {
	// Asegúrate de que la partición montada no sea nula
	if partition == nil {
		return fmt.Errorf("Error: La partición montada proporcionada es nula")
	}

	// Abrir el archivo de disco para leer el Superblock
	file, err := Utilities.OpenFile(partition.Path)
	if err != nil {
		return fmt.Errorf("Error al abrir el archivo de disco: %v", err)
	}
	defer file.Close()

	// Leer el MBR para ubicar la partición
	var TempMBR Structs.MBR
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		return fmt.Errorf("Error al leer el MBR del archivo: %v", err)
	}

	// Buscar la partición correspondiente dentro del MBR
	var partitionData Structs.Partition
	partitionFound := false
	for i := 0; i < 4; i++ {
		if string(TempMBR.Partitions[i].Id[:]) == partition.ID {
			partitionData = TempMBR.Partitions[i]
			partitionFound = true
			break
		}
	}

	// Si no se encuentra la partición dentro del MBR, retornar un error
	if !partitionFound {
		return fmt.Errorf("Error: Partición no encontrada dentro del MBR")
	}

	// Leer el Superblock de la partición
	var superblock Structs.Superblock
	superblockOffset := int64(partitionData.Start)
	if err := Utilities.ReadObject(file, &superblock, superblockOffset); err != nil {
		return fmt.Errorf("Error al leer el Superblock desde el archivo: %v", err)
	}

	// Crear las carpetas padre si no existen
	if err := createDirectoryIfNotExists(path); err != nil {
		return fmt.Errorf("Error al crear directorios: %v", err)
	}

	// Obtener el nombre base del archivo sin la extensión y la imagen de salida
	dotFileName, outputImage := getFileNames(path)

	// Iniciar el contenido DOT para el Superbloque
	dotContent := `digraph G {
		node [shape=plaintext];
		tabla [label=<
			<table border="1" cellborder="1" cellspacing="0" cellpadding="10">
				<tr><td colspan="2" bgcolor="darkgreen"><font color="white">Reporte de SUPERBLOQUE</font></td></tr>
				<tr><td bgcolor="lightgreen">sb_nombre_hd</td><td>` + partition.Path + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_filesystem_type</td><td>` + fmt.Sprintf("%d", superblock.S_filesystem_type) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_inodes_count</td><td>` + fmt.Sprintf("%d", superblock.S_inodes_count) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_blocks_count</td><td>` + fmt.Sprintf("%d", superblock.S_blocks_count) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_free_blocks_count</td><td>` + fmt.Sprintf("%d", superblock.S_free_blocks_count) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_free_inodes_count</td><td>` + fmt.Sprintf("%d", superblock.S_free_inodes_count) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_mtime</td><td>` + cleanDateString(string(superblock.S_mtime[:])) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_umtime</td><td>` + cleanDateString(string(superblock.S_umtime[:])) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_mnt_count</td><td>` + fmt.Sprintf("%d", superblock.S_mnt_count) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_magic</td><td>` + fmt.Sprintf("0x%X", superblock.S_magic) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_inode_size</td><td>` + fmt.Sprintf("%d", superblock.S_inode_size) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_block_size</td><td>` + fmt.Sprintf("%d", superblock.S_block_size) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_fist_ino</td><td>` + fmt.Sprintf("%d", superblock.S_fist_ino) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_first_blo</td><td>` + fmt.Sprintf("%d", superblock.S_first_blo) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_bm_inode_start</td><td>` + fmt.Sprintf("%d", superblock.S_bm_inode_start) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_bm_block_start</td><td>` + fmt.Sprintf("%d", superblock.S_bm_block_start) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_inode_start</td><td>` + fmt.Sprintf("%d", superblock.S_inode_start) + `</td></tr>
				<tr><td bgcolor="lightgreen">sb_block_start</td><td>` + fmt.Sprintf("%d", superblock.S_block_start) + `</td></tr>
			</table>>];
	}`

	// Crear el archivo DOT
	dotFilePath := filepath.Join(path, dotFileName)
	if err := os.WriteFile(dotFilePath, []byte(dotContent), 0644); err != nil {
		return fmt.Errorf("Error al crear o escribir en el archivo DOT: %v", err)
	}

	// Ejecutar el comando Graphviz para generar la imagen en la misma carpeta
	outputImagePath := filepath.Join(path, outputImage)
	if err := exec.Command("dot", "-Tpng", dotFilePath, "-o", outputImagePath).Run(); err != nil {
		return fmt.Errorf("Error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Imagen del reporte del Superbloque generada en:", outputImagePath)
	return nil
}

func GenerateFileReport(path string, partition MountedPartition, pathFileLs string) {
	// Implementación pendiente
}

func GenerateLsReport(path string, partition MountedPartition, pathFileLs string) {
	// Implementación pendiente
}

// GetMountedPartitionSuperblock busca una partición montada por su ID y obtiene su Superblock.
func GetMountedPartitionSuperblock(id string) (*Structs.Superblock, *MountedPartition, string, error) {
	// Buscar la partición montada con el ID proporcionado
	var mountedPartition *MountedPartition
	var partitionFound bool

	// Buscar en el mapa de particiones montadas
	for _, partitions := range mountedPartitions {
		for _, partition := range partitions {
			if partition.ID == id {
				mountedPartition = &partition
				partitionFound = true
				break
			}
		}
		if partitionFound {
			break
		}
	}

	// Si no se encontró la partición, retornar un error
	if !partitionFound {
		return nil, nil, "", fmt.Errorf("Error: Partición con ID %s no encontrada", id)
	}

	// Abrir el archivo de disco para leer el Superblock
	file, err := Utilities.OpenFile(mountedPartition.Path)
	if err != nil {
		return nil, nil, "", fmt.Errorf("Error al abrir el archivo de disco: %v", err)
	}
	defer file.Close()

	// Leer el MBR para ubicar la partición
	var TempMBR Structs.MBR
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		return nil, nil, "", fmt.Errorf("Error al leer el MBR del archivo: %v", err)
	}

	// Buscar la partición correspondiente dentro del MBR
	var partition Structs.Partition
	partitionFound = false
	for i := 0; i < 4; i++ {
		if string(TempMBR.Partitions[i].Id[:]) == mountedPartition.ID {
			partition = TempMBR.Partitions[i]
			partitionFound = true
			break
		}
	}

	// Si no se encuentra la partición dentro del MBR, retornar un error
	if !partitionFound {
		return nil, nil, "", fmt.Errorf("Error: Partición no encontrada dentro del MBR")
	}

	// Leer el Superblock de la partición
	var superblock Structs.Superblock
	superblockOffset := int64(partition.Start)
	if err := Utilities.ReadObject(file, &superblock, superblockOffset); err != nil {
		return nil, nil, "", fmt.Errorf("Error al leer el Superblock desde el archivo: %v", err)
	}

	// Retornar el Superblock, la partición montada y la ruta del archivo
	return &superblock, mountedPartition, mountedPartition.Path, nil
}
