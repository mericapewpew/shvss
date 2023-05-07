#!/bin/bash

program_name="shvss"
service_file="/etc/systemd/system/${program_name}.service"

if (( $EUID == 0 )); then
    echo "Please run as user"
    exit
fi
user_name=$(whoami)
function build() {
    CGO_ENABLED=0 go build -o ${program_name} -v -ldflags="-s -w" .
}

function scriptInstaller() {
    echo "#!/bin/bash
sudo systemctl stop ${program_name}.service
sudo systemctl disable ${program_name}.service
sudo rm /usr/bin/${program_name}
sudo rm /etc/systemd/system/${program_name}.service
sudo rm /usr/share/${program_name} -rf
sudo rm /usr/bin/${program_name}-uninstaller
echo '${program_name} removed'" > ${program_name}-uninstaller
    sudo mv ${program_name}-uninstaller /usr/bin
    sudo chmod +x /usr/bin/${program_name}-uninstaller
    echo "To remove this program a script was installed:'
    ${program_name}-uninstaller
run this to remove all files and directories"
}

function installService() {
    printf "Port Number:"
    portNumber=$(read var; echo $var)
    echo "[Unit]
Description=${program_name}
After=network.target

[Service]
Type=simple
User=${user_name}
WorkingDirectory=/usr/share/${program_name}
ExecStart=/usr/bin/${program_name}  -p ${portNumber} -s /usr/share/${program_name}/subs.json
Restart=always

[Install]
WantedBy=multi-user.target"  > ${program_name}.service
    sudo mv ${program_name}.service /etc/systemd/system/
    sudo systemctl enable ${program_name}.service
    sudo systemctl start ${program_name}.service
}

function installBinary() {
    build
    sudo mv ${program_name} /usr/bin/
}

function makeDataDir() {
    sudo mkdir /usr/share/${program_name}
    sudo touch /usr/share/${program_name}/subs.json
    sudo chown -R ${user_name}:${user_name} /usr/share/${program_name}
}

function install() {
    installBinary
    makeDataDir
    installService
    scriptInstaller
}

function update() {
    if [[ -f "${service_file}" ]]; then
        git pull
        sudo systemctl stop ${program_name}.service
        installBinary
        sudo systemctl start ${program_name}.service
    else
        echo "${program_name} not installed
        bash installer.sh install"
    fi
}

case "$1" in
    "install")
        install;;
    "update")
        update;;
    "remove")
        /usr/bin/${program_name}-uninstaller;;
    "build")
        build;;
    *)
        printf "%s\n" "bash installer.sh <arg>
    install
    update
    remove
    build
    ";;
esac
