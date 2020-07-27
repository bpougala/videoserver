# videoserver
## An HLS bite-sized video streaming server written in Go

### The background

Video streaming is one of today's most popular ways of using the Internet. If photos, documents and short videos can be sent as one file over HTTP, this is not the case for larger videos (either in length or in quality). HLS (HTTP Live Streaming) is a video streaming protocol originally developed by Apple that allows the server to cut a large file into small chunks (a few MBs in size each) and send them individually over the Internet. First the video player reads a manifest file (with extension `.m3u8` which hosts the names and locations of every chunk), then the video player will query the server for each individual chunk. This way the player can start the video without having received the full file and if the server connection fails, the player can just query for the last chunk instead of querying for the whole file. Another advantage of HLS is *adaptive bitate streaming*. This allows the video player to automatically select a video quality when more than one is available based on the network conditions. 

### The project

This is a small project that fetches a video stored in an AWS S3 bucket in the form of a manifest file and one or more chunks, then accepts GET requests for either the manifest file (with `fetchVideoM3U8FileFromS3()` and `serveHlsM3u8FileFromS3()`) or each individual chunk (with `fetchVideoChunkS3()` and `serveHlsTsFromS3()`). 

