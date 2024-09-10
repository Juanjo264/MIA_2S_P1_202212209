mkdisk -size=2 -unit=M -fit=WF -path="/home/juanjo/disks/DiscoLab.mia"
fdisk -size=300 -type=P -unit=K -fit=B -name="Particion1" -path="/home/juanjo/disks/DiscoLab.mia"
fdisk -size=100 -type=P -unit=K -fit=F -name="Particion2" -path="/home/juanjo/disks/DiscoLab.mia"
fdisk -size=100 -type=E -unit=K -fit=B -name="ParticionE" -path="/home/juanjo/disks/DiscoLab.mia" 
fdisk -size=50 -type=L -unit=K -fit=B -name="ParticionL" -path="/home/juanjo/disks/DiscoLab.mia"
mount -name="Particion1" -path="/home/juanjo/disks/DiscoLab.mia"
mkfs -id=091a -type=full
rep -id=091a -path="/home/juanjo/output/report_mbr.png" -name=mbr
rep -id=091a -path="/home/juanjo/output/report_inode.png" -name=inode
rep -id=091a -path="/home/juanjo/output/report_disk.png" -name=disk




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
#rep
rep -id=091a -path="/home/juanjo/output/report_mbr.png" -name=mbr
rep -id=091a -path="/home/juanjo/output/report_inode.png" -name=inode
rep -id=091a -path="/home/juanjo/output/report_disk.png" -name=disk
#fin de comandos