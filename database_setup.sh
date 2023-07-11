#!/bin/bash


# Reset
Color_Off='\033[0m'       # Text Reset
ResetC=$Color_Off

# Regular Colors
Black='\033[0;30m'        # Black
Red='\033[0;31m'          # Red
Green='\033[0;32m'        # Green
Yellow='\033[0;33m'       # Yellow
Blue='\033[0;34m'         # Blue
Purple='\033[0;35m'       # Purple
Cyan='\033[0;36m'         # Cyan
White='\033[0;37m'        # White

print_color() {
    prt=$1
    color=$2
    echo -e "${color}${prt}${ResetC}"
}

print_red() {
    print_color "$1" "$Red"
}

print_yellow() {
    print_color "$1" "$Yellow"
}

print_green() {
    print_color "$1" "$Green"
}

check_sudo() {
    if [[ $EUID -ne 0 ]]
    then
    print_yellow "Setup script needs to be run as 'root'. Please use 'sudo setup.sh'"
    exit 1
    fi
}

# ---------------------------------------------------------------
# Main
# ---------------------------------------------------------------

echo "Starting database setup"

check_sudo

# Finding if mysql database is installed and install in case it is not
if ! command -v mysql &> /dev/null
then
    print_red "Mysql not installed!"
    print_yellow "Installing Mysql..."
    nala -y install mariadb-server
fi

# Setup
logfile="database.log"
# Creating a new database
mysql < "mariadb_setup.sql" | tee "$logfile"

print_green "All preparations steps for the database performed."
print_green "You are ready to go :)"

# ---------------------------------------------------------------
# Main End
# ---------------------------------------------------------------