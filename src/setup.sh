#!/bin/bash

# A setup script for driveraker

echo "Making config directory..."
mkdir $HOME/.config/driveraker

echo "Downloading configuration file templates...\n"
mkdir /tmp/driveraker
cd /tmp/driveraker
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/docs/examples/conf.json $HOME/.config/driveraker/config

echo "Downloading systemd timer and service file...\n"
mkdir -p $HOME/.config/systemd/user/
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/src/systemd/driveraker.service $HOME/.config/systemd/user/driveraker.service
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/src/systemd/driveraker.timer $HOME/.config/systemd/user/driveraker.timer

echo "Downloading scripts for driveraker...\n"
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/src/systemd/sync.sh $HOME/.config/driveraker/sync
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/src/copyHugoSite.sh $HOME/.config/driveraker/copyHugoSite.sh

echo "Downloading driveraker binary...\n"
wget https://github.com/gatlinnewhouse/driveraker/releases/download/untagged-20996afe1b3b1d9c73e9/driveraker $HOME/.config/driveraker/driveraker

echo "\n"
echo "Make sure to have the following installed:\n"
echo "* systemd\n"
echo "* pandoc with version >= 1.19.2.1\n"
echo "* go with version >= 1.8\n"
echo "* drive (https://github.com/odeke-em/drive)\n"
echo "* hugo framework\n"
echo "* an http(s) server like nginx or apache\n"
echo "\n"

echo "[IMPORTANT] Make sure to replace USERNAME with your username in the configuration file!\n"
echo "[IMPORTANT] Make sure to make note of your DriveSyncDirectory, and HugoPostDirectory!\n\n"
read -p "Press ENTER to edit your driveraker configuration or press Ctrl+C to cancel:"; echo

nano $HOME/.config/driveraker/config

echo "\n"
echo "Enter your DriveSyncDirectory path:\n"
drivesyncdirectory="$inputline"
cd $drivesyncdirectory
drive init

echo "\n"
echo "Enter your HugoPostDirectory:\n"
hugositedirectory="$inputline"
hugo new site $hugositedirectory

echo "\n"
echo "You should be all setup now :)"
