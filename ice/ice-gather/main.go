package main

import (
	"fmt"
	"log"

	"github.com/nkbai/goice/ice"
)

func main() {
	addrs, err := ice.DefaultGatherer.Gather()
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to gather: %s", err))
	}
	for _, a := range addrs {
		fmt.Printf("%s\n", a)
		//laddr, err := net.ResolveUDPAddr("udp",
		//	a.ZeroPortAddr(),
		//)
		//if err != nil {
		//	log.Fatal(err)
		//}
		//c, err := net.ListenUDP("udp", laddr)
		//if err != nil {
		//	fmt.Println("   ", "failed:", err)
		//	continue
		//}
		//fmt.Println("   ", "bind ok", c.LocalAddr())
		//c.Close()
	}
}
