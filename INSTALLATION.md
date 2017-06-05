# Installing prerequisites

On an Ubuntu 17.04 server.

## Installing nginx

First, install nginx:

```bash
sudo apt install nginx
```

Enable the firewall:

```bash
sudo ufw enable
```

Check the firewall rules:

```bash
sudo ufw app list
```

Which nets the following:

```bash
Available applications:
  Nginx Full
  Nginx HTTP
  Nginx HTTPS
  OpenSSH
```

Since I am running a server without HTTPS right now, I only will enable Nginx HTTP:

```bash
sudo ufw allow 'Nginx HTTP'
```

Now you can run `sudo ufw status` to see if the rules updated.

Then check if nginx is running properly by running `systemctl status nginx.service`, the results should be something like:

```bash
● nginx.service - A high performance web server and a reverse proxy server
   Loaded: loaded (/lib/systemd/system/nginx.service; enabled; vendor preset: enabled)
   Active: active (running) since Mon 2017-06-05 17:39:22 UTC; 4min 30s ago
     Docs: man:nginx(8)
  Process: 8917 ExecStart=/usr/sbin/nginx -g daemon on; master_process on; (code=exited, status=0/SUCCESS)
  Process: 8905 ExecStartPre=/usr/sbin/nginx -t -q -g daemon on; master_process on; (code=exited, status=0/SUCCESS)
 Main PID: 8922 (nginx)
    Tasks: 3 (limit: 4915)
   Memory: 8.2M
      CPU: 44ms
   CGroup: /system.slice/nginx.service
           ├─8922 nginx: master process /usr/sbin/nginx -g daemon on; master_process on;
           ├─8923 nginx: worker process
           └─8924 nginx: worker process
```

Then visit `http://<YOUR-SERVER-IP>:80/` in your web-browser. You should see a page with "Welcome to nginx!" in a bold font.

## Installing [drive](https://github.com/odeke-em/drive)

Install go with `sudo apt install golang` then set your gopath:

```bash
cat << ! >> ~/.bashrc
> export GOPATH=\$HOME/gopath
> export PATH=\$GOPATH:\$GOPATH/bin:\$PATH
> !
source ~/.bashrc # To reload the settings and get the newly set ones # Or open a fresh terminal
```

Then run `sudo add-apt-repository ppa:twodopeshaggy/drive` to add a PPA for drive.

Run `sudo apt update` and then `sudo apt install drive` to install drive.

### Setup drive with credentials

Make a directory for syncing with `mkdir ~/gdrive` and then `cd ~/gdrive` into the new directory.

Next run `drive init` and follow instructions given.

## Installing pandoc

```bash
sudo apt install pandoc
```

## Installing [hugo](https://github.com/spf13/hugo)

```bash
sudo apt install hugo
```
