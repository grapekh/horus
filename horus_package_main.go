/*
 * Copyright (c) 2018 by Howard I Grapek <howiegrapek@yahoo.com>
 *
 * GPL License:
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 *
 * Howie's Notes...
 * This is basically a network scanner - like nmap - written in GO language
 * First piece - just a stub to prove connectivity and compile environment. 
 *
 *
 * Description:
 * This will ultimately be the miner check and report program
 * The normal port to check for miners will be: 4028
 * 
 * For one of my Decred Miners, I found this to verify the port:
 *
 * C:\Users\howie\Apps\Nmap>echo {"command":"version"} | ncat 10.0.0.5 4028
 * {"STATUS":[{"STATUS":"S","When":1532052885,"Code":22,"Msg":"CGMiner versions","Description":"sgminer 4.4.2"}],"VERSION":[{"CGMiner":"4.4.2","API":"3.4"}],"id":1}
 * C:\Users\howie\Apps\Nmap>
 *
 * USAGE: 
	Usage: horus.exe [-m value ...]
		-m   	0 or more Miner addresses - can be mixture of CIDR blocks or IP addresses
	
		Note, If no value is specified for -m, the current local network will be searched +
		for any/all miners on the subnet.

 *
 * REQUIRED LIBRARIES: 
 * cgminer.go - Used public API to break apart responses from miners. 
 * Currently in: c:\users\howie\go\src\cgminer-api
 *
 * Version History: 
 * 
 * 0.1 - grapek - original start to walk network
 * 0.2 - grapek - find actual miners
 * 0.3 - grapek - use command line to accept ip, cidr block or default to local area network
 * 0.4 - grapek - actually connect and display some information from the miners. (use test stubs)

package main

import (
	"fmt"
	"log"
	"net"
	"time"
	"sync"
	"os"
	"strings"
	"cgminer-api"			// Howie's Reqired Package. 
)


type MyNet struct {
	IP           *net.IPNet			// CIDR Block - complete address
	MyIp         net.IP			// x.x.x.x
	Netmask      net.IPMask			// ffffff00
	Subnet       net.IP			// first ip address of network block based on netmask
	AvailableIPs []string          		// list of unique addresses of those which have miners on them
}


// Global Constants and Variables. 

var Horus_Version string = "Version 0.04"
var date = time.Now()	
var date_string = date.Format("Mon Jan 02 2006 at 15:04:05")


// Is the scanner remote to the network or on the
// internal network? (TODO: Howie: Futures)
var remote = false

// create a global empty slice of strings
// and a bool for debugging
var ips []string
var ports []string
var debug bool = false


// A WaitGroup waits for a collection of goroutines to finish. The main loop calls Add(1)
// to set the number of goroutines to wait for. Then each of the goroutines runs and calls Done
// when finished.
var wg sync.WaitGroup

// A channel that acts as a counting semaphore. Only allow X concurrent connections.
// The number here can be adjusted up or down. If too many open files/sockets then
// adjust this down. Lower numbers mean slower scan times.
// `ls /proc/pidof netscan/fd | wc -l` should be just under this
var sem = make(chan struct{}, 32768)

const usage string =
	"\n\nUsage: horus.exe [-m value ...]\n" +
		"-m   	0 or more Miner addresses - can be mixture of CIDR blocks or IP addresses\n" +
		"\n" +
		"Note, If no value is specified for -m, the current local network will be searched \n" +
		"for any/all miners on the subnet.\n"


const connection_timeout = 2 * time.Second
//const connection_timeout = 20 * time.Millisecond

//////////////////////////////////////////////////////////////
// Get my external interface -
// This will work even if there are multiple interfaces to the existing machine.
//////////////////////////////////////////////////////////////
func resolveHostIp() (string) {
	netInterfaceAddresses, err := net.InterfaceAddrs()
	if err != nil { return "" }

	for _, netInterfaceAddress := range netInterfaceAddresses {
		networkIp, ok := netInterfaceAddress.(*net.IPNet)

		if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() != nil {
			ip := networkIp.IP.String()
			fmt.Println("DEBUG: Resolved Host IP: " + ip)
			return ip
		}
	}
	return ""
}

//////////////////////////////////////////////////////////////
// Get connected and plumbed local area network information
/////////////////////////////////////////////////////////////
func getMyLanInfo() (*MyNet) {
	mn := new(MyNet)
	ifaces, err := net.Interfaces()

	// Handle any error (there shouldn't be
	if err != nil {
		log.Fatal(err)
	}

	// Loop through all the network interfaces -
	// skip any down interfaces and loopback
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}

		addrs, err := iface.Addrs()
		if err != nil {
			log.Fatal(err)
		}

		// Walk the interfaces and get real info.
		for _, addr := range addrs {
			var ip net.IP				// The ip address of my public network interface
			var netmask net.IPMask		// My netmask

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
				netmask = v.Mask

			case *net.IPAddr:
				ip = v.IP
				continue
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			// Fill the structure with details.
			mn.IP = addr.(*net.IPNet)		// The Cidr Block
			mn.MyIp = ip                    // my IPv4 Address
			mn.Netmask = netmask            // My Netmask
			mn.Subnet = ip.Mask(netmask)    // My Subnet

			// Show me what I found on my network.
			fmt.Println("\nMy Local Network Information:")
			fmt.Printf("Cidr Block ..... (%s)\n", mn.IP)
			fmt.Printf("IPv4 Address ... (%s)\n", mn.MyIp)
			fmt.Printf("Netmask ........ (%s)\n", mn.Netmask)
			fmt.Printf("Subnet ......... (%s)\n\n", mn.Subnet)
		}
	}

	return mn
}


/////////////////////////////////////////////////////////////
// connect
// Take an IP string and a ptr to the ports slice
//
// Check to see if we can connect to a specific port 
// If we can, report on it and add to global list of assets 
/////////////////////////////////////////////////////////////
func connect(ip string, the_ports []string, m *MyNet) {

	for _, port := range the_ports {
		if debug {
			fmt.Fprintln(os.Stderr, "checking port: ", port, "on ip: ", ip)
		}

		hostPort := net.JoinHostPort(ip, port)

		conn, err := net.DialTimeout("tcp", hostPort, connection_timeout)
		if err != nil {
			if debug {
				fmt.Printf("Cannot connect to %s on port %s - error: %s\n", ip, port, err.Error())
			}
		} else {
			fmt.Printf(" ... Success on Port: %s - IP: %s\n", port, ip)
		
			// Add ip address to list of good ones in our global structure. - Only add if unique and not found already
			// (TODO: Howie) this may not work for concurrency, I need to check that there isn't already added
			m.AvailableIPs = AppendIfMissing(m.AvailableIPs, ip)

			conn.Close()
		}
	}

	// Decrements wg counter by 1
	defer wg.Done()

	// Receive Signal
	<-sem
}

/////////////////////////////////////////////////////////////
// Unique Append:
// Append the string to a slice only if it is not there already
/////////////////////////////////////////////////////////////
func AppendIfMissing(slice []string, s string) []string {
    for _, ele := range slice {
        if ele == s {
            return slice
        }
    }
    return append(slice, s)
}

/////////////////////////////////////////////////////////////
// Break apart the cidr block into its individual ip addresses
/////////////////////////////////////////////////////////////
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}


////  
// testing stubs
////

func Test_Summary(miner_ip string) {
	miner := cgminer.New(miner_ip, 4028)
	summary, err := miner.Summary()
	if err != nil {
		fmt.Println("Got an error back from miner.Summary: ", err)
		return
	}
	if summary == nil {
		fmt.Println("Summary returned nil")
		return
	}

	fmt.Printf("Found Blocks: %d\n", summary.FoundBlocks)
	fmt.Printf("Accepted: %d\n", summary.Accepted)
	fmt.Printf("Rejected: %d\n", summary.Rejected)

	//fmt.Printf("Status: %s\n", summary.Status)
}


func Test_Devs(miner_ip string) {
	miner := cgminer.New(miner_ip, 4028)
	devs, err := miner.Devs()
	if err != nil {
		fmt.Println("Got an error back from miner.Devs: ", err)
		return
	}
	if devs == nil {
		fmt.Println("Devs returned nil")
		return
	}
	for _, dev := range *devs {
		fmt.Printf("Dev %d temp: %f\n", dev.GPU, dev.Temperature)
	}
}


func Test_Pools(miner_ip string) {
	miner := cgminer.New(miner_ip, 4028)
	pools, err := miner.Pools()
	if err != nil {
		fmt.Println("Got an error back from miner.Pools: ", err)
		return
	}
	for _, pool := range pools {
		fmt.Printf("Pool %d: %s\n", pool.Pool, pool.URL)
	}
}


//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func main() {

	var pips []string                   // temporary list of IP's
	var my_cidr string                  // My CIDR Block as discovered. 

	// Shortcut for println
	p := fmt.Println

	fmt.Printf("HORUS (%s): Starting on %s\n ", Horus_Version, date_string)

	// Get local area info, If we don't give any networks on the commmand line - use this network as a default
	MyLanInfo := getMyLanInfo()
	my_cidr = fmt.Sprintf("%s", MyLanInfo.IP)

	if debug {
		p("MyCidr after conversion is: ", my_cidr)
	}

	// Parse Commandline Arguments. 
	// can be -help or -m 
	// any other items on the command line are considered 1 or more ip addresses/cidr blocks. 
	// args[0] is the name of the program, so we don't count that. 

    if len (os.Args) == 1 {
    	// No command line arguments specified 
	   	p("Searching for miners on all IP's in local area network...")

    	//pips = []string{"172.22.70.80", "172.22.201.231", "172.22.201.231/24", "172.22.201.228"}
    	pips = append(pips, my_cidr)
    } else {
    	// Parse Command line args. 

		for i, arg := range os.Args {

			if i == 0 {
				continue                 // ignore the argv[0] - the program name
			} 

			if arg == "-help" {
				fmt.Println (usage)
				time.Sleep(2 * time.Second)
				os.Exit(1)
			}

			if arg == "-m" {
				p("Searching for miners on ip's or cidr blocks entered on command line...\n")
			} else {
				if debug {
					fmt.Printf("arg %d: %s\n", i, os.Args[i])
				}
				pips = append(pips, os.Args[i])
			}
		}
	}
	
	
	// Ports to check - for testing, we can test multiple ports. 
	// Note, for miners, the port required is only CGMiner: 4028, 
	// for other nmap like operations, we can search for ssh, http, etc
	//ports := []string{"21", "22", "80", "4028"}
	ports := []string{"4028"}

	if debug {
		fmt.Println("Ports being checked are: ", ports)
	}

	if debug {
		fmt.Println("IPs being checked are: ", pips)

		for _, ip := range pips {
			fmt.Fprintf(os.Stderr, "In for loop ... Checking IP: %s\n", ip)
		}
	}

	//
	// Just some diagnostics and debug output:
	//
	if debug {
		fmt.Println("\nAt bottom of main - before getting network... the myLanInfo struct is:")
		fmt.Println(MyLanInfo)
		fmt.Printf("ipv4 .... (%s)\n", MyLanInfo.IP)
	}

	//
	// MAIN NMAP FUNCTION TO FIND ALL THE HOSTS LOOKING FOR MINERS.
	// I can make this a function, but the amount of code and the fact that it is called only once 
	// makes it justtifyable that it is inline. 
	//

	// Set up the variable to use herein
	var ipnet 	*net.IPNet						// The IP Network address to check (individual ip or cidr block)
	var newip 	net.IP							// Single IP to
	var err   	error							// errors

	// Traverse the list of IPs passed in (either my network, or commandline ip's)
	for _, ip := range pips {
		if debug {
			fmt.Fprintln(os.Stderr, "In for loop ... Checking IP %s", ip)
		}

		// Lets see if there is a CIDR Block or not.
		if !strings.Contains(ip, "/") {
			// We got an ip not a CIDR block. no prob. just process it.
			newip := net.ParseIP(ip)

			if debug {
				fmt.Println("after parsing - we have an ip, not a cidr block: ", newip)
			}

			wg.Add(1)									// Add 1 to wg counter
			sem <- struct{}{}							// Send Signal into channel
			go connect(ip, ports, MyLanInfo)			// Do the nmap scan.
			continue

		}

		newip, ipnet, err = net.ParseCIDR(ip)
		if err != nil {
			fmt.Println("Got an error back from net.Parse: ", err)
		} else {
			if debug {
				fmt.Println("In a CIDR Block: ip:", ip, "newip", newip, "ipnet: ", ipnet)
			}

			// walk the network block with the appropriate mask.
			for newip := newip.Mask(ipnet.Mask); ipnet.Contains(newip); inc(newip) {

				if debug {
					fmt.Println("Inside a Cidr Block Walking network with this ip: ", newip)
				}

				wg.Add(1)										// Add 1 to wg counter
				sem <- struct{}{}								// Send Signal into channel
				go connect(newip.String(), ports, MyLanInfo)	// Check the individual IP with all ports in list.
			}
		}
	}

	// blocks until the WaitGroup counter is zero.
	wg.Wait()
	close(sem)

	// Ok, now we know exactly what we are working with, how many miners we have, 
	// and can grab those ip's out of the global memory when needed. 

	fmt.Println("\n\nComplete list of ip addresses who are miners:")

	num_miners := 0
	for _, ip := range MyLanInfo.AvailableIPs {
		fmt.Fprintf(os.Stderr, " ... IP: %s\n", ip)
		num_miners++
	}

	fmt.Printf("Total Number of unique miners found: %d\n", num_miners) 

	// Lets get some details from the miners (if any)
	//for _, ip := range MyLanInfo.AvailableIPs {

			ip := "10.0.0.5"

			fmt.Printf("Getting summary information for: %s\n", ip)
			Test_Summary(ip)

			fmt.Printf("Getting dev information for: %s\n", ip)
			Test_Devs(ip)
			
			fmt.Printf("Getting pool information for: %s\n", ip)
			Test_Pools(ip)
	//}

}
