# Installing G-Wan http(s) server

Install dependencies first:

```bash
sudo apt install sqlite3 libtokyocabinet-dev libtokyocabinet9 tokyocabinet-bin gobjc++ golang
```

Then make the appropriate symbolic links and folders for g-wan:

```bash
sudo mkdir --parents /usr/local/go/bin
sudo ln --symbolic /usr/bin/go /usr/local/go/bin/go
```

Then [install (and *optimally configure*) g-wan](http://gwan.com/download):

```bash
wget http://gwan.com/archives/gwan_linux64-bit.tar.bz2
tar -xjf gwan_linux64-bit.tar.bz2; cd gwan_linux64-bit
./gwan                       (./gwan -h for help)
```

Then type `http://<YOUR-SERVER-IP>:8080/` into your broswer and [play with the settings of g-wan...](http://gwan.com/download)
