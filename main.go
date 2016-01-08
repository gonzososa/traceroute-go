package main

import (
          "fmt"
          "os"
          "net"
          "time"
          "golang.org/x/net/ipv4"
)

// Proto define udp protocol type
const (
          Protocol      = "udp4"
          ICMP          = "ip4:icmp"
          LocalAddress  = "0.0.0.0"
          Port          = 33434
          BufferSize    = 128
          MaxTTL        = 30
          Deadline      = 3
)

// Check for possible errors
func Check (err error) {
          if err != nil  {
                  panic (err.Error ())
          }
}

// SendPacket to hostname with incremental ttl
func SendPacket (hostname string, addresses chan string, exit chan bool) {
        remoteAddress, err := net.ResolveUDPAddr (Protocol, fmt.Sprintf ("%s:%d", hostname, Port))
        Check (err)

        conn, err := net.DialUDP (Protocol, nil, remoteAddress)
        Check (err)
        defer conn.Close ()

        packet := ipv4.NewPacketConn (conn)
        defer packet.Close ()
        var ttl = 1
        var output, destAddress string

        for (destAddress != remoteAddress.IP.String () && ttl <= MaxTTL) {
                packet.SetTTL (ttl)

                buffer := make ([]byte, 0x00)
                _, err := packet.Write (buffer) // write bytes through connected socket
                Check (err)

                destAddress = <- addresses //wait for routers response

                if destAddress == "" {
                        output = fmt.Sprintf ("%d\t%s\t%s", ttl, "*", "*")
                } else {
                        output = fmt.Sprintf ("%d\t%s", ttl, destAddress)
                }

                fmt.Println (output)
                ttl++
        }

        exit <- true
}

// ListenEcho from routers in path
func ListenEcho (addresses chan string) {
        localAddress, err := net.ResolveIPAddr (ICMP, LocalAddress)
        Check (err)

        c, err := net.ListenIP (ICMP, localAddress)
        Check (err)
        c.SetReadDeadline (time.Now().Add (time.Second * Deadline))
        defer c.Close ()

        for {
                buffer := make ([]byte, BufferSize)
                bytesRead, remoteAddress, err := c.ReadFromIP (buffer)
                if e, ok := err.(net.Error); ok && e.Timeout() {
                        // if err were a timeout we don't raise panic
                        c.SetReadDeadline (time.Now().Add (time.Second * Deadline))
                        addresses <- ""
                        continue
                } else {
                        Check (err)
                }

                if bytesRead > 0 {
                        addresses <- remoteAddress.String ()
                }
        }
}

func main () {
        if len (os.Args) <= 1 { return }

        hostname := os.Args [1]
        exit := make (chan bool)
        addresses := make (chan string)

        go ListenEcho (addresses)
        go SendPacket (hostname, addresses, exit)

        <- exit
}
