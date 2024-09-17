package Analyzer

import (
	"backend/DiskManagement"
	"backend/FileSystem"
	"backend/User"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`-(\w+)=("[^"]+"|\S+)`)

func Analyzer(input string) (string, error) {
	// Divide la entrada en tokens usando espacios en blanco como delimitadores
	tokens := strings.Fields(input)

	// Si no se proporcionó ningún comando, devuelve un error
	if len(tokens) == 0 {
		return "", errors.New("no se proporcionó ningún comando")
	}

	switch tokens[0] {
	case "mkdisk":
		return fn_mkdisk(tokens[1:])
	case "rmdisk":
		return fn_rmdisk(tokens[1:])
	case "fdisk":
		return fn_fdisk(tokens[1:])
	case "mount":
		return fn_mount(tokens[1:])
	case "mkfs":
		return fn_mkfs(tokens[1:])
	case "login":
		return fn_login(tokens[1:])
	case "rep":
		return fn_rep(tokens[1:])
	case "logout":
		return User.Logout()
	case "mkfile":
		return FileSystem.ParserMkfile(tokens[1:])
	case "mkdir":
		return fn_mkdir(tokens[1:])
	case "clear":
		// Crea un comando para limpiar la terminal
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout // Redirige la salida del comando a la salida estándar
		err := cmd.Run()       // Ejecuta el comando
		if err != nil {
			// Si hay un error al ejecutar el comando, devuelve un error
			return "", errors.New("no se pudo limpiar la terminal")
		}
		return "", nil // Devuelve nil si el comando se ejecutó correctamente
	default:
		// Si el comando no es reconocido, devuelve un error
		return "", fmt.Errorf("comando desconocido: %s", tokens[0])
	}
}

func fn_mkdisk(tokens []string) (string, error) {
	fs := flag.NewFlagSet("mkdisk", flag.ExitOnError)
	size := fs.Int("size", 0, "Tamaño")
	fit := fs.String("fit", "ff", "Ajuste")
	unit := fs.String("unit", "m", "Unidad")
	path := fs.String("path", "", "Ruta")

	// Parse flag
	fs.Parse(os.Args[1:])

	// Encontrar la flag en el input
	matches := re.FindAllStringSubmatch(strings.Join(tokens, " "), -1)

	// Process the input
	for _, match := range matches {
		flagName := match[1]                   // match[1]: Captura y guarda el nombre del flag (por ejemplo, "size", "unit", "fit", "path")
		flagValue := strings.ToLower(match[2]) // Captura y guarda el valor del flag

		flagValue = strings.Trim(flagValue, "\"")

		switch flagName {
		case "size", "fit", "unit":
			flagValue = strings.ToLower(flagValue)
			fs.Set(flagName, flagValue)
		case "path":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag not found")
		}
	}
	/*
			Primera Iteración :
		    flagName es "size".
		    flagValue es "3000".
		    El switch encuentra que "size" es un flag reconocido, por lo que se ejecuta fs.Set("size", "3000").
		    Esto asigna el valor 3000 al flag size.

	*/

	// Validaciones
	if *size <= 0 {
		fmt.Println("Error: Size must be greater than 0")
		return "", fmt.Errorf("parámetro inválido: %v", *size)

	}

	if *fit != "bf" && *fit != "ff" && *fit != "wf" {
		fmt.Println("Error: Fit must be 'bf', 'ff', or 'wf'")
		return "", fmt.Errorf("parámetro inválido: %s", *fit)

	}

	if *unit != "k" && *unit != "m" {
		fmt.Println("Error: Unit must be 'k' or 'm'")
		return "", fmt.Errorf("parámetro inválido: %s", *unit)

	}

	if *path == "" {
		fmt.Println("Error: Path is required")
		return "", fmt.Errorf("parámetro inválido: %s", *path)

	}

	// LLamamos a la funcion
	// Llamar a la función Mkdisk y capturar el mensaje de éxito
	message, err := DiskManagement.Mkdisk(*size, *fit, *unit, *path)
	if err != nil {
		return "", err
	}
	return message, nil
}

func fn_rmdisk(tokens []string) (string, error) {
	fs := flag.NewFlagSet("rmdisk", flag.ExitOnError)
	path := fs.String("path", "", "Ruta del disco a eliminar")

	// Parse flag
	fs.Parse(os.Args[1:])

	// Encontrar el flag en el input
	matches := re.FindAllStringSubmatch(strings.Join(tokens, " "), -1)

	// Procesar los parámetros del comando
	for _, match := range matches {
		flagName := match[1]
		flagValue := strings.Trim(match[2], "\"")

		if flagName == "path" {
			fs.Set(flagName, flagValue)
		} else {
			fmt.Println("Error: Flag no encontrado:", flagName)
		}
	}

	// Validar el parámetro de la ruta
	if *path == "" {
		fmt.Println("Error: Path es requerido")
		return "", fmt.Errorf("parámetro inválido: %s", *path)
	}

	// Llamar a la función que elimina el disco y capturar su retorno
	message, err := DiskManagement.Rmdisk(*path)
	if err != nil {
		// Retornar el mensaje y error de la función Rmdisk
		return message, err
	}

	return message, nil
}

func fn_fdisk(tokens []string) (string, error) {
	// Definir flags
	fs := flag.NewFlagSet("fdisk", flag.ExitOnError)
	size := fs.Int("size", 0, "Tamaño")
	path := fs.String("path", "", "Ruta")
	name := fs.String("name", "", "Nombre")
	unit := fs.String("unit", "m", "Unidad")
	type_ := fs.String("type", "p", "Tipo")
	fit := fs.String("fit", "", "Ajuste") // Dejar fit vacío por defecto

	// Parsear los flags
	fs.Parse(os.Args[1:])

	// Encontrar los flags en el input
	matches := re.FindAllStringSubmatch(strings.Join(tokens, " "), -1)

	// Procesar el input
	for _, match := range matches {
		flagName := match[1]
		flagValue := strings.ToLower(match[2])

		flagValue = strings.Trim(flagValue, "\"")

		switch flagName {
		case "size", "fit", "unit", "path", "name", "type":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag not found")
		}
	}

	// Validaciones
	if *size <= 0 {
		fmt.Println("Error: Size must be greater than 0")
		return ``, fmt.Errorf("parámetro inválido: %v", *size)
	}

	if *path == "" {
		fmt.Println("Error: Path is required")
		return ``, fmt.Errorf("parámetro inválido: %s", *path)
	}

	// Si no se proporcionó un fit, usar el valor predeterminado "w"
	if *fit == "" {
		*fit = "w"
	}

	// Validar fit (b/w/f)
	if *fit != "b" && *fit != "f" && *fit != "w" {
		fmt.Println("Error: Fit must be 'b', 'f', or 'w'")
		return ``, fmt.Errorf("parámetro inválido: %s", *fit)
	}

	if *unit != "k" && *unit != "m" {
		fmt.Println("Error: Unit must be 'k' or 'm'")
		return ``, fmt.Errorf("parámetro inválido: %s", *unit)
	}

	if *type_ != "p" && *type_ != "e" && *type_ != "l" {
		fmt.Println("Error: Type must be 'p', 'e', or 'l'")
		return ``, fmt.Errorf("parámetro inválido: %s", *type_)
	}

	// Llamar a la función
	message, err := DiskManagement.Fdisk(*size, *path, *name, *unit, *type_, *fit)
	if err != nil {
		return "", err
	}
	return message, nil
}

func fn_mount(tokens []string) (string, error) {
	fs := flag.NewFlagSet("mount", flag.ExitOnError)
	path := fs.String("path", "", "Ruta")
	name := fs.String("name", "", "Nombre de la partición")

	// Parsear los argumentos
	fs.Parse(os.Args[1:])
	matches := re.FindAllStringSubmatch(strings.Join(tokens, " "), -1)

	// Procesar los parámetros del comando
	for _, match := range matches {
		flagName := match[1]
		flagValue := strings.ToLower(match[2]) // Convertir todo a minúsculas
		flagValue = strings.Trim(flagValue, "\"")
		fs.Set(flagName, flagValue)
	}

	// Validar los parámetros requeridos
	if *path == "" || *name == "" {
		fmt.Println("Error: Path y Name son obligatorios")
		return "", fmt.Errorf("parámetro inválido: %s", *path)
	}

	// Convertir el nombre a minúsculas antes de pasarlo al Mount
	lowercaseName := strings.ToLower(*name)

	// Llamar a la función de montaje y capturar su retorno
	message, err := DiskManagement.Mount(*path, lowercaseName)
	if err != nil {
		return message, err
	}

	// Retornar el mensaje de éxito de la función Mount
	return message, nil
}

func fn_mkfs(tokens []string) (string, error) {
	fs := flag.NewFlagSet("mkfs", flag.ExitOnError)
	id := fs.String("id", "", "Id")
	type_ := fs.String("type", "", "Tipo")
	fs_ := fs.String("fs", "2fs", "Fs")

	// Parse the input string, not os.Args
	matches := re.FindAllStringSubmatch(strings.Join(tokens, " "), -1)

	for _, match := range matches {
		flagName := match[1]
		flagValue := match[2]

		flagValue = strings.Trim(flagValue, "\"")

		switch flagName {
		case "id", "type", "fs":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag not found")
		}
	}

	// Verifica que se hayan establecido todas las flags necesarias
	if *id == "" {
		fmt.Println("Error: id es un parámetro obligatorio.")
		return "", fmt.Errorf("parámetro inválido: %s", *id)
	}

	if *type_ == "" {
		fmt.Println("Error: type es un parámetro obligatorio.")
		return "", fmt.Errorf("parámetro inválido: %s", *type_)
	}

	// Llamar a la función
	message, err := FileSystem.Mkfs(*id, *type_, *fs_)
	if err != nil {
		return "", err
	}
	return message, nil
}

func fn_login(tokens []string) (string, error) {
	// Definir las flags
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	user := fs.String("user", "", "Usuario")
	pass := fs.String("pass", "", "Contraseña")
	id := fs.String("id", "", "Id")

	// Parsearlas
	fs.Parse(os.Args[1:])

	// Match de flags en el input
	matches := re.FindAllStringSubmatch(strings.Join(tokens, " "), -1)

	// Procesar el input
	for _, match := range matches {
		flagName := match[1]
		flagValue := match[2]

		flagValue = strings.Trim(flagValue, "\"")

		switch flagName {
		case "user", "pass", "id":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag not found")
		}
	}

	message, err := User.Login(*user, *pass, *id)
	if err != nil {
		return "", err
	}
	return message, nil
}

func fn_rep(tokens []string) (string, error) {
	// Definir flags para el comando rep
	fs := flag.NewFlagSet("rep", flag.ExitOnError)
	name := fs.String("name", "", "Nombre del reporte a generar (mbr, disk, inode, block, bm_inode, bm_block, sb, file, ls)")
	path := fs.String("path", "", "Ruta donde se guardará el reporte")
	id := fs.String("id", "", "ID de la partición que se utilizará")
	pathFileLs := fs.String("path_file_ls", "", "Nombre del archivo o carpeta para los reportes 'file' y 'ls'")

	// Parsear los flags
	fs.Parse(os.Args[1:])
	// Match de flags en el input
	matches := re.FindAllStringSubmatch(strings.Join(tokens, " "), -1)

	// Procesar los parámetros del input
	for _, match := range matches {
		flagName := match[1]
		flagValue := match[2]
		flagValue = strings.Trim(flagValue, "\"")

		switch flagName {
		case "name", "path", "id", "path_file_ls":
			fs.Set(flagName, flagValue)
		default:
			fmt.Println("Error: Flag not found")
			return "", fmt.Errorf("parámetro inválido: %s", flagName)
		}
	}

	// Validar los parámetros obligatorios
	if *name == "" {
		fmt.Println("Error: El parámetro -name es obligatorio y debe contener un valor válido (mbr, disk, inode, block, bm_inode, bm_block, sb, file, ls)")
		return "", fmt.Errorf("parámetro inválido: %s", *name)
	}

	if *path == "" {
		fmt.Println("Error: El parámetro -path es obligatorio.")
		return "", fmt.Errorf("parámetro inválido: %s", *path)
	}

	if *id == "" {
		fmt.Println("Error: El parámetro -id es obligatorio.")
		return "", fmt.Errorf("parámetro inválido: %s", *id)
	}

	// Verificar que el nombre del reporte es válido
	validReports := []string{"mbr", "disk", "inode", "block", "bm_inode", "bm_block", "sb", "file", "ls"}
	if !isValidReportName(*name, validReports) {
		fmt.Println("Error: Nombre de reporte no válido.")
		return "", fmt.Errorf("parámetro inválido: %s", *name)
	}

	// Verificar que la partición con el id existe
	partition := DiskManagement.GetPartitionByID(*id)
	if partition == nil {
		fmt.Println("Error: No se encontró la partición con el id proporcionado.")
		return "", fmt.Errorf("partición no encontrada: %s", *id)
	}

	// Generar el reporte con Graphviz
	switch *name {
	case "mbr":
		DiskManagement.GenerateMBRReport(*path, *partition)
	case "disk":
		DiskManagement.GenerateDiskReport(*path, partition)
	case "inode":
		DiskManagement.GenerateInodeReport(*path, partition)

	case "block":
		DiskManagement.GenerateBlockReport(*path, *partition)
	case "bm_inode":
		DiskManagement.GenerateBMInodeReport(*path, *partition)
	case "bm_block":
		DiskManagement.GenerateBMBlockReport(*path, *partition)
	case "sb":
		DiskManagement.GenerateSuperblockReport(*path, *partition)
	case "file":
		if *pathFileLs == "" {
			fmt.Println("Error: El parámetro -path_file_ls es obligatorio para el reporte file.")
			return "", fmt.Errorf("parámetro inválido: %s", *pathFileLs)
		}
		DiskManagement.GenerateFileReport(*path, *partition, *pathFileLs)
	case "ls":
		if *pathFileLs == "" {
			fmt.Println("Error: El parámetro -path_file_ls es obligatorio para el reporte ls.")
			return "", fmt.Errorf("parámetro inválido: %s", *pathFileLs)
		}
		DiskManagement.GenerateLsReport(*path, *partition, *pathFileLs)
	default:
		fmt.Println("Error: Nombre de reporte no válido.")
	}
	return "REP: Reporte " + *name + " exitosamente en: " + *path, nil
}

// Verifica si el nombre del reporte es válido
func isValidReportName(name string, validReports []string) bool {
	for _, report := range validReports {
		if name == report {
			return true
		}
	}
	return false
}

func fn_mkdir(tokens []string) (string, error) {
	// Parsear los argumentos del comando mkdir
	args := make(map[string]string)
	for _, token := range tokens {
		matches := re.FindStringSubmatch(token)
		if len(matches) == 3 {
			args[matches[1]] = strings.Trim(matches[2], `"`)
		}
	}

	// Verificar que se haya proporcionado el argumento -path
	path, ok := args["path"]
	if !ok {
		return "", errors.New("no se proporcionó el argumento -path")
	}

	// Llamar a la función Mkdir para crear los directorios
	logs, err := FileSystem.Mkdir(path)
	if err != nil {
		return logs, err
	}

	return logs, nil
}
