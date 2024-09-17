mkdisk -size=2 -unit=M -fit=WF -path="/home/juanjo/disks/DiscoLab.mia"
rmdisk -path="/home/juanjo/disks/discolab.mia"
fdisk -size=300 -type=P -unit=K -fit=B -name="Particion1" -path="/home/juanjo/disks/DiscoLab.mia"
fdisk -size=100 -type=P -unit=K -fit=F -name="Particion2" -path="/home/juanjo/disks/DiscoLab.mia"
fdisk -size=100 -type=E -unit=K -fit=B -name="ParticionE" -path="/home/juanjo/disks/DiscoLab.mia" 
fdisk -size=50 -type=L -unit=K -fit=B -name="ParticionL" -path="/home/juanjo/disks/DiscoLab.mia"
fdisk -size=50 -type=L -unit=K -fit=B -name="ParticionL2" -path="/home/juanjo/disks/DiscoLab.mia"

mount -name="Particion2" -path="/home/juanjo/disks/DiscoLab.mia"
mount -name="Particion1" -path="/home/juanjo/disks/DiscoLab.mia"
mkfs -id=091a -type=full

login -user=root -pass=123 -id=091a 
login -user="mi usuario" -pass="mi pwd" -id=091a 

mkdir -path="/home"
mkdir -path="/home/usac"
mkdir -path="/home/work"
mkdir -path="/home/usac/mia"

rep -id=091a -path="/home/juanjo/output/report_mbr.png" -name=mbr
rep -id=091a -path="/home/juanjo/output/report_disk.png" -name=disk

rep -id=091a -path="/home/juanjo/output/report_inode.png" -name=inode
mkfile -size=68 -path=/home/usac/mia/a.txt

logout



rep -id=091A -path="/home/juanjo/Descargas/MIA_2S_P1_202212209_1/output/report_bm_inode.txt" -name=bm_inode

{
  "command": "mkdisk -size=2 -unit=M -fit=WF -path=\"/"/home/juanjo/disks/DiscoLab.mia""
}


#inicio
mkdisk -size=2 -unit=M -fit=WF -path="/home/juanjo/disks/DiscoLab.mia"
fdisk -size=300 -type=P -unit=K -fit=B -name="Particion1" -path="/home/juanjo/disks/DiscoLab.mia"
fdisk -size=100 -type=P -unit=K -fit=F -name="Particion2" -path="/home/juanjo/disks/DiscoLab.mia"
fdisk -size=100 -type=E -unit=K -fit=B -name="ParticionE" -path="/home/juanjo/disks/DiscoLab.mia" 
fdisk -size=50 -type=L -unit=K -fit=B -name="ParticionL" -path="/home/juanjo/disks/DiscoLab.mia"
mount -name="Particion1" -path="/home/juanjo/disks/DiscoLab.mia"
mkfs -id=091a -type=full
login -user=root -pass=123 -id=091a 
login -user="mi usuario" -pass="mi pwd" -id=091a 
mkdir -path="/home"
mkdir -path="/home/usac"
mkdir -path="/home/work"
mkdir -path="/home/usac/mia"
mkfile -size=68 -path=/home/usac/mia/a.txt
#rep
rep -id=091a -path="/home/juanjo/output/report_mbr.png" -name=mbr
rep -id=091a -path="/home/juanjo/output/report_inode.png" -name=inode
rep -id=091a -path="/home/juanjo/output/report_disk.png" -name=disk
logout

#fin de comandos

mkdisk -size=2 -unit=K -fit=WF -path="/home/juanjo/disks/DiscoLab.mia"




