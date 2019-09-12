/*
   Alien Invasion Simulator
*/

package main

import (
	"fmt"
   "os"
   "strconv"
)

func printHelp() {
     fmt.Println();
     fmt.Println("Map generation mode usage: ");
     fmt.Println("   ais -gen <MAPFILE> <MAXX> <MAXY> <CD> <RD>");
     fmt.Println();
     fmt.Println("   <MAPFILE>  Name of the output file where the generated map data will be stored.");
     fmt.Println("   <MAXX>     Positive integer width of the city grid.");
     fmt.Println("   <MAXY>     Positive integer height of the city grid..");
     fmt.Println("   <CD>       Real number in the [0, 1] range for the density of cities in the grid.");
     fmt.Println("   <RD>       Real number in the [0, 1] range for the density of roads in the grid.");
     fmt.Println();
     fmt.Println();
     fmt.Println("Simulation mode usage: ");
     fmt.Println("   ais <MAPFILE> <NUMALIENS>");
     fmt.Println();
     fmt.Println("   <MAPFILE>    Name of the input file where the generated map data is stored.");
     fmt.Println("   <NUMALIENS>  Positive integer number of aliens to unleash in the city.");
     fmt.Println();
}

func generate(mapfile string, maxx int, maxy int, cd float64, rd float64) {
     fmt.Printf("TODO: Will write mapfile '%s' with dimensions %d x %d, city density %f and road density %f.\n", mapfile, maxx, maxy, cd, rd);
}

func simulate(mapfile string, numaliens int) {
     fmt.Printf("TODO: Will read mapfile '%s' and simulate it with %d aliens.\n", mapfile, numaliens);
}

func main() {
	fmt.Println("Alien Invasion Simulator!\n")

   if (len(os.Args) < 2) {
      fmt.Println("No arguments given.");
      printHelp();
   } else if (os.Args[1] == "-gen") {
      if (len(os.Args) < 7) {
         fmt.Println("Too few arguments for map generation mode.");
         printHelp();
      } else if (len(os.Args) > 7) {
         fmt.Printf("Too many arguments for map generation mode: '%s'.\n", os.Args[7]);
         printHelp();
      } else {
        mapfile := os.Args[2];
        maxx, ok := strconv.Atoi( os.Args[3] );
        maxy, ok := strconv.Atoi( os.Args[4] );
        cd, ok := strconv.ParseFloat( os.Args[5], 64 );
        rd, ok := strconv.ParseFloat( os.Args[6], 64 );
        if (ok != nil) {
           fmt.Println("Generate: Error parsing numeric arguments.");
           printHelp();
        } else {
          generate(mapfile, maxx, maxy, cd, rd);
        }
      }
   } else if (os.Args[1][0] == '-') {
     fmt.Printf("Unsupported command: '%s'\n", os.Args[1]);
     printHelp();
   } else {
      if (len(os.Args) < 3) {
         fmt.Println("Too few arguments for simulation mode.");
         printHelp();
      } else if (len(os.Args) > 3) {
         fmt.Printf("Too many arguments for simulation mode: '%s'.\n", os.Args[3]);
         printHelp();
      } else {
        mapfile := os.Args[1];
        numaliens, ok := strconv.Atoi( os.Args[2] );
        if (ok != nil) {
           print("Simulate: Error parsing numeric arguments.");
           printHelp();
        } else {
           simulate(mapfile, numaliens);
        }
      }
   }
}
