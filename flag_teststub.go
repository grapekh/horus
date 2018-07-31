package main

import (
	"flag"
	"fmt"
	"os"
	"net"
)

const version string = "V1.06"
var Debug bool = false

func Test_Cmdline_IP(iplist []string)  bool {

	for _, ipl := range iplist {
		if Debug {
			fmt.Printf("Search the following for miners: %s\n", ipl)
		}

		trial := net.ParseIP(ipl)

		if trial.To4() != nil {
			if Debug {
        			fmt.Printf("%v is a valid IPv4 address ... continuing to next check\n", trial)
        	}
        	continue
        }

		if debug {
        	fmt.Printf("Maybe it is a cidrblock? \n")
        }

		ipA,ipnetA,_ := net.ParseCIDR(ipl)
        
        if dDbug {
    			fmt.Println("ipA              : ", ipA)
    			fmt.Println("ipnetA           : ", ipnetA)
    	}

		if ipA.To4() != nil {
			if Debug {
        		fmt.Printf("%v is a valid IPv4 address as part of a cidr block ... continuing to next check\n", ipA)
        	}
        	continue
        }

        // do we have a fatal error? 
        fmt.Println("Error: Network address specified on command line: (", ipl, ") is not a valid IP address or CIDR block.  Exiting.")
        os.Exit(1)
    }



	return true
}

func main() {

	
	usage := `
	Horus: The Local Area Network Miner Explorer:

	Display information for any ASIC Miners found on the local area network. 
	You may specify 0 or more IPs, CIDR blocks or a mixture of each on the command line.
	If no arguments are specified, your current class C network will be searched. 

	USAGE:
		horus (-v | -h) | [-d] [ < IP OR CIDR_BLOCK > ...]

	USAGE EXAMPLES: 
		horus                       Displays information for all miners discovered on the current class C Network
		horus -d 10.0.4.2           Display the miner info on 1.0.4.2 with debug output
		horus -v                    Displays program version and exist
		horus -h                    Display this help message and exit
		horus 10.0.2.12             Display information for miner found on ip adddress 10.0.2.12
		horus 10.0.2.0/24           Display information for miners found in the cidr block network 10.0.2.0/24
		horus 10.2.4.6 10.2.4.0/24  Display information for miners found in the list of ip or cidr blocks provided
	                                                 (note, you can mix and match ip and cidr blocks)
	OPTIONS:
		-v       Displays the version of the program and exists
		-h       Display this message and exits
		-d       optional: Add Debug output to miner details   
`

	// The options
	verPtr := flag.Bool("v", false, "Display Program Version and Exit")
	hlpPtr := flag.Bool("h", false, "Display Usage and Exit")
	dbgPtr := flag.Bool("d", false, "Add DEBUG information to results")

	flag.Parse()

	// Deal with the options

	if *verPtr == true {
		fmt.Println("Horus: Version ", version)
		os.Exit(0)
	}

	if *hlpPtr == true {
		fmt.Println(usage)
		os.Exit(0)
	}

	if *dbgPtr == true {
		Debug = true
	}


 	//
	// Deal with the list of IP's specified - 
	// TODO - Validate that any/all ip's are either IP or CIDR block format. 
	//

	if Debug {
		fmt.Println("IP / CIDR Blocks Specified:", flag.Args())
	}

	//ipl := flag.Args()
	// Lets validate the arguments. 
	if Test_Cmdline_IP(flag.Args()) {
		fmt.Println("Commandlines Are All Good!!!", flag.Args())
	}
	
    // We're good here - lets process and continue. 
    if len(flag.Args()) > 0 {
   		fmt.Println("All addresses are valid - continuing to parse...")
    	fmt.Println("IP / CIDR Blocks Specified:", flag.Args())
    } else {
    	fmt.Println("No commandline arguments specified - using current LAN.")
    }

}

