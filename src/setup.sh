#!/bin/bash

# A setup script for driveraker

echo "Making config directory..."
mkdir $HOME/.config/driveraker

echo "Downloading configuration file templates..."
mkdir /tmp/driveraker
cd /tmp/driveraker
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/docs/examples/conf.json $HOME/.config/driveraker/config

echo "Downloading systemd timer and service file..."
mkdir -p $HOME/.config/systemd/user/
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/src/systemd/driveraker.service $HOME/.config/systemd/user/driveraker.service
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/src/systemd/driveraker.timer $HOME/.config/systemd/user/driveraker.timer

echo "Downloading scripts for driveraker..."
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/src/systemd/sync.sh $HOME/.config/driveraker/sync
wget https://raw.githubusercontent.com/gatlinnewhouse/driveraker/master/src/copyHugoSite.sh $HOME/.config/driveraker/copyHugoSite.sh

echo "Downloading driveraker binary..."
wget https://github.com/gatlinnewhouse/driveraker/releases/download/untagged-20996afe1b3b1d9c73e9/driveraker $HOME/.config/driveraker/driveraker

echo
echo "Make sure to have the following installed:"
echo "* systemd"
echo "* pandoc with version >= 1.19.2.1"
echo "* go with version >= 1.8"
echo "* drive (https://github.com/odeke-em/drive)"
echo "* hugo framework"
echo "* an http(s) server like nginx or apache"
echo

echo "[IMPORTANT] Make sure to replace USERNAME with your username in the configuration file!"
echo "[IMPORTANT] Make sure to make note of your DriveSyncDirectory, and HugoPostDirectory!"
echo
read -p "Press ENTER to edit your driveraker configuration or press Ctrl+C to cancel:"; echo

nano $HOME/.config/driveraker/config

echo
read -p "Enter your DriveSyncDirectory path: " drivesyncdirectory
cd $drivesyncdirectory
drive init

echo
read -p "Enter your HugoPostDirectory: " hugositedirectory
hugo new site $hugositedirectory

echo
echo "You should be all setup now :)"
echo "Make sure to run \"systemctl enable driveraker.service\" and \"systemctl enable driveraker.timer\" for systemd support."
