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

// Mapa para almacenar las particiones montadas, organizadas por disco
var mountedPartitions = make(map[string][]MountedPartition)

// Función para imprimir las particiones montadas
func PrintMountedPartitions() {
	fmt.Println("Particiones montadas:")

	if len(mountedPartitions) == 0 {
		fmt.Println("No hay particiones montadas.")
		return
	}

	for diskID, partitions := range mountedPartitions {
		fmt.Printf("Disco ID: %s\n", diskID)
		for _, partition := range partitions {
			loginStatus := "No"
			if partition.LoggedIn {
				loginStatus = "Sí"
			}
			fmt.Printf(" - Partición Name: %s, ID: %s, Path: %s, Status: %c, LoggedIn: %s\n",
				partition.Name, partition.ID, partition.Path, partition.Status, loginStatus)
		}
	}
	fmt.Println("")
}

// Función para obtener las particiones montadas
func GetMountedPartitions() map[string][]MountedPartition {
	return mountedPartitions
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
	fmt.Println("======INICIO MKDISK======")
	fmt.Println("Size:", size)
	fmt.Println("Fit:", fit)
	fmt.Println("Unit:", unit)
	fmt.Println("Path:", path)

	// Validar fit bf/ff/wf
	if fit != "bf" && fit != "wf" && fit != "ff" {
		fmt.Println("Error: Fit debe ser bf, wf or ff")
		return "Error: Fit debe ser bf, wf or ff", nil

	}

	// Validar size > 0
	if size <= 0 {
		fmt.Println("Error: Size debe ser mayo a  0")
		return "Error: Size debe ser mayo a  0", nil

	}

	// Validar unidar k - m
	if unit != "k" && unit != "m" {
		fmt.Println("Error: Las unidades validas son k o m")
		return "Error: Las unidades validas son k o m", nil

	}

	// Create file
	err := Utilities.CreateFile(path)
	if err != nil {
		fmt.Println("Error: ", err)
		return "error", nil

	}

	/*
		Si el usuario especifica unit = "k" (Kilobytes), el tamaño se multiplica por 1024 para convertirlo a bytes.
		Si el usuario especifica unit = "m" (Megabytes), el tamaño se multiplica por 1024 * 1024 para convertirlo a MEGA bytes.
	*/
	// Asignar tamanio
	if unit == "k" {
		size = size * 1024
	} else {
		size = size * 1024 * 1024
	}

	// Open bin file
	file, err := Utilities.OpenFile(path)
	if err != nil {
		return "", nil

	}

	// Escribir los 0 en el archivo

	// create array of byte(0)
	for i := 0; i < size; i++ {
		err := Utilities.WriteObject(file, byte(0), int64(i))
		if err != nil {
			fmt.Println("Error: ", err)
		}
	}

	// Crear MRB
	var newMRB Structs.MRB
	newMRB.MbrSize = int32(size)
	newMRB.Signature = rand.Int31() // Numero random rand.Int31() genera solo números no negativos
	copy(newMRB.Fit[:], fit)

	// Obtener la fecha del sistema en formato YYYY-MM-DD
	currentTime := time.Now()
	formattedDate := currentTime.Format("2006-01-02")
	copy(newMRB.CreationDate[:], formattedDate)

	/*
		newMRB.CreationDate[0] = '2'
		newMRB.CreationDate[1] = '0'
		newMRB.CreationDate[2] = '2'
		newMRB.CreationDate[3] = '4'
		newMRB.CreationDate[4] = '-'
		newMRB.CreationDate[5] = '0'
		newMRB.CreationDate[6] = '8'
		newMRB.CreationDate[7] = '-'
		newMRB.CreationDate[8] = '0'
		newMRB.CreationDate[9] = '8'
	*/

	// Escribir el archivo
	if err := Utilities.WriteObject(file, newMRB, 0); err != nil {
		return "", nil

	}

	var TempMBR Structs.MRB
	// Leer el archivo
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		return "", nil

	}

	// Print object
	Structs.PrintMBR(TempMBR)

	// Cerrar el archivo
	defer file.Close()

	fmt.Println("======FIN MKDISK======")

	return fmt.Sprintf("MKDISK: Disco creado exitosamente en: %s", path), nil
}

func Fdisk(size int, path string, name string, unit string, type_ string, fit string) (string, error) {
	fmt.Println("======Start FDISK======")
	fmt.Println("Size:", size)
	fmt.Println("Path:", path)
	fmt.Println("Name:", name)
	fmt.Println("Unit:", unit)
	fmt.Println("Type:", type_)
	fmt.Println("Fit:", fit)

	// Validar fit (b/w/f)
	if fit != "b" && fit != "f" && fit != "w" {
		fmt.Println("Error: Fit must be 'b', 'f', or 'w'")
		return "Error: Fit must be 'b', 'f', or 'w'", nil
	}

	// Validar size > 0
	if size <= 0 {
		fmt.Println("Error: Size must be greater than 0")
		return "Error: Size must be greater than 0", nil
	}

	// Validar unit (b/k/m)
	if unit != "b" && unit != "k" && unit != "m" {
		fmt.Println("Error: Unit must be 'b', 'k', or 'm'")
		return "Error: Unit must be 'b', 'k', or 'm'", nil
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
		fmt.Println("Error: Could not open file at path:", path)
		return "Error: Could not open file at path", nil
	}

	var TempMBR Structs.MRB
	// Leer el objeto desde el archivo binario
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		fmt.Println("Error: Could not read MBR from file")
		return "Error: Could not read MBR from file", nil
	}

	// Imprimir el objeto MBR
	Structs.PrintMBR(TempMBR)

	fmt.Println("-------------")

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
		fmt.Println("Error: No se pueden crear más de 4 particiones primarias o extendidas en total.")
		return "Error: No se pueden crear más de 4 particiones primarias o extendidas en total", nil
	}

	// Validar que solo haya una partición extendida
	if type_ == "e" && extendedCount > 0 {
		fmt.Println("Error: Solo se permite una partición extendida por disco.")
		return "Error: Solo se permite una partición extendida por disco", nil
	}

	// Validar que no se pueda crear una partición lógica sin una extendida
	if type_ == "l" && extendedCount == 0 {
		fmt.Println("Error: No se puede crear una partición lógica sin una partición extendida.")
		return "Error: No se puede crear una partición lógica sin una partición extendida", nil
	}

	// Validar que el tamaño de la nueva partición no exceda el tamaño del disco
	if usedSpace+int32(size) > TempMBR.MbrSize {
		fmt.Println("Error: No hay suficiente espacio en el disco para crear esta partición.")
		return "Error: No hay suficiente espacio en el disco para crear esta partición", nil
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

				// Imprimir el nuevo EBR creado
				fmt.Println("Nuevo EBR creado:")
				Structs.PrintEBR(newEBR)
				fmt.Println("")

				// Imprimir todos los EBRs en la partición extendida
				fmt.Println("Imprimiendo todos los EBRs en la partición extendida:")
				ebrPos = TempMBR.Partitions[i].Start
				for {
					err := Utilities.ReadObject(file, &ebr, int64(ebrPos))
					if err != nil {
						fmt.Println("Error al leer EBR:", err)
						break
					}
					Structs.PrintEBR(ebr)
					if ebr.PartNext == -1 {
						break
					}
					ebrPos = ebr.PartNext
				}

				break
			}
		}
		fmt.Println("")
	}

	// Sobrescribir el MBR
	if err := Utilities.WriteObject(file, TempMBR, 0); err != nil {
		fmt.Println("Error: Could not write MBR to file")
		return "Error: Could not write MBR to file", nil
	}

	var TempMBR2 Structs.MRB
	// Leer el objeto nuevamente para verificar
	if err := Utilities.ReadObject(file, &TempMBR2, 0); err != nil {
		fmt.Println("Error: Could not read MBR from file after writing")
		return "Error: Could not read MBR from file after writing", nil
	}

	// Imprimir el objeto MBR actualizado
	Structs.PrintMBR(TempMBR2)

	// Cerrar el archivo binario
	defer file.Close()

	fmt.Println("======FIN FDISK======")
	fmt.Println("")
	return fmt.Sprintf("FDISK: Partición "+name+" creada exitosamente en: %s", path), nil
}

// Función para montar particiones
func Mount(path string, name string) (string, error) {
	file, err := Utilities.OpenFile(path)
	if err != nil {
		fmt.Println("Error: No se pudo abrir el archivo en la ruta:", path)
		return "Error: No se pudo abrir el archivo en la ruta", nil
	}
	defer file.Close()

	var TempMBR Structs.MRB
	if err := Utilities.ReadObject(file, &TempMBR, 0); err != nil {
		fmt.Println("Error: No se pudo leer el MBR desde el archivo")
		return "Error: No se pudo leer el MBR desde el archivo", nil
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
		fmt.Println("Error: Partición no encontrada o no es una partición primaria")
		return "Error: Partición no encontrada o no es una partición primaria", nil
	}

	// Verificar si la partición ya está montada
	if partition.Status[0] == '1' {
		fmt.Println("Error: La partición ya está montada")
		return "Error: La partición ya está montada", nil
	}

	//fmt.Printf("Partición encontrada: '%s' en posición %d\n", string(partition.Name[:]), partitionIndex+1)

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
	carnet := "202212209" // Cambiar su carnet aquí
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
		fmt.Println("Error: No se pudo sobrescribir el MBR en el archivo")
		return "Error: No se pudo sobrescribir el MBR en el archivo", nil
	}

	fmt.Printf("Partición montada con ID: %s\n", partitionID)

	fmt.Println("")
	// Imprimir el MBR actualizado
	fmt.Println("MBR actualizado:")
	Structs.PrintMBR(TempMBR)
	fmt.Println("")

	// Imprimir las particiones montadas (solo estan mientras dure la sesion de la consola)
	PrintMountedPartitions()
	return fmt.Sprintf("Partición montada con ID: %s", partitionID), nil
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

	var mbr Structs.MRB
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

	var mbr Structs.MRB
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
	err := createDirectoryIfNotExists(path)
	if err != nil {
		return err
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
	superblockOffset := int64(binary.Size(Structs.MRB{})) // Ajusta según la posición real del Superblock
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

		// Convertir tiempos a string
		atime := cleanDateString(string(inode.I_atime[:]))
		ctime := cleanDateString(string(inode.I_ctime[:]))
		mtime := cleanDateString(string(inode.I_mtime[:]))

		// Definir el contenido DOT para el inodo actual con colores
		dotContent += fmt.Sprintf(`inode%d [label=<
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
			`, i, i, inode.I_uid, inode.I_gid, inode.I_size, atime, ctime, mtime, cleanString(string(inode.I_type[:])), string(inode.I_perm[:]))

		// Agregar los bloques directos a la tabla
		for j, block := range inode.I_block[:12] {
			color := "#FFFFFF" // Color de fondo para bloques asignados
			if block == -1 {
				color = "#FFCCCB" // Resaltar bloques no asignados en rojo claro
			}
			dotContent += fmt.Sprintf("<tr><td bgcolor=\"#F5F5F5\">Bloque %d</td><td bgcolor=\"%s\">%d</td></tr>", j+1, color, block)
		}

		// Agregar bloques indirectos, doble y triple con colores distintos
		dotContent += fmt.Sprintf(`
			<tr><td colspan="2" bgcolor="#D3D3D3"><b>BLOQUE INDIRECTO</b></td></tr>
			<tr><td bgcolor="#F5F5F5">13</td><td bgcolor="#FFFFFF">%d</td></tr>
			<tr><td colspan="2" bgcolor="#D3D3D3"><b>BLOQUE INDIRECTO DOBLE</b></td></tr>
			<tr><td bgcolor="#F5F5F5">14</td><td bgcolor="#FFFFFF">%d</td></tr>
			<tr><td colspan="2" bgcolor="#D3D3D3"><b>BLOQUE INDIRECTO TRIPLE</b></td></tr>
			<tr><td bgcolor="#F5F5F5">15</td><td bgcolor="#FFFFFF">%d</td></tr>
			</table>>];
		`, inode.I_block[12], inode.I_block[13], inode.I_block[14])

		// Conectar con el siguiente inodo si no es el último
		if i < superblock.S_inodes_count-1 {
			dotContent += fmt.Sprintf("inode%d -> inode%d;\n", i, i+1)
		}
	}

	// Cerrar el contenido DOT
	dotContent += "}"

	// Crear el archivo DOT
	dotFilePath := filepath.Join(path, dotFileName)
	fileDot, err := os.Create(dotFilePath)
	if err != nil {
		return fmt.Errorf("error al crear el archivo DOT: %v", err)
	}
	defer fileDot.Close()

	// Escribir el contenido DOT en el archivo
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

	fmt.Println("Imagen del reporte de inodos generada en:", outputImagePath)
	return nil
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

func GenerateBlockReport(path string, partition MountedPartition) {
	// Implementación pendiente
}

func GenerateBMInodeReport(path string, partition MountedPartition) {
	// Implementación pendiente
}

func GenerateBMBlockReport(path string, partition MountedPartition) {
	// Implementación pendiente
}

func GenerateSuperblockReport(path string, partition MountedPartition) {
	// Implementación pendiente
}

func GenerateFileReport(path string, partition MountedPartition, pathFileLs string) {
	// Implementación pendiente
}

func GenerateLsReport(path string, partition MountedPartition, pathFileLs string) {
	// Implementación pendiente
}
