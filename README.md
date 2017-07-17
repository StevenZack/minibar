## Introduction
minibar is a simple DFS example project written in Go . Our goal is to combine multiple storage devices into one , 
and make them as easy-to-use as one storage device on the cloud.

## Usage
#### 1.Download the release <a href='http://github.com/stevenzack/minibar/releases/latest'>here</a>,and uncompress it<br>
<pre><code>$tar xvf minibar_linux_amd64.tar.bz2</code></pre>
#### 2.Setup a master server (who controls all the traffic , handle all request, but never store any files)<br>
<pre><code>$./minibar -p 8090 master</code></pre>
><b>-p</b>  refers to the port you want your master server to listen
<b>master</b> means this is a master server

#### 3.Add 2 volume servers:volume1 and volume2 , then connect them to the master server<br>
<pre><code>$./minibar -p 8080 --max 100000000 --mserver localhost:8090 --dir ./v1 volume<br>
$./minibar -p 8081 --max 100000000 --mserver localhost:8090 --dir ./v1 volume</code></pre><br>
><b>-p</b>  refers to the port you want your volume server to listen<br>
<b>--max</b>  refers to the max disk space you wanna use in the volume device<br>
<b>--mserver</b> your master server IP you want to connect<br>
<b>--dir</b> directory for saving files<br>
<b>volume</b> means this is a volume server
***
#### 4.
