/*
   Alien Invasion Simulator
*/

package main

import (
	"fmt"
   "os"
   "strconv"
   "math/rand"
   "time"
   "bufio"
   "strings"
)

// ---------------------------------------------------------------------------------------------------
// Map generator data model
// ---------------------------------------------------------------------------------------------------

// The map generator produces a regular 2D grid of nodes, where each node may or
//   may not contain a city.

// Indices for Node.roads
const EAST  int = 0;
const SOUTH int = 1;

// A node in our map data model.
// NORTH and WEST roads can be obtained by reading the SOUTH and EAST roads
//   of the previous node.
//
type Node struct {
     cityName    string    // Name of the city in this node, or "" if none.
     roads       [2]bool   // Outbound roads in this node: EAST and SOUTH 
}

// A world map.
type World [][]Node;

// ---------------------------------------------------------------------------------------------------
// Simulator data model
// ---------------------------------------------------------------------------------------------------

// The simulator does not assume that the provided input file conforms to any topological
//   constraints, so its data model is different from the generator's simple model.
// Each city node (SNode) that we read in has pointers for other city nodes that lie in the
//   four cardinal directions. The only assumption we make is that if city A has a "north"
//   connection to city B, then city B has a "south" connection to city A (and similar to east-west
//   roads). If the input file violates that (e.g. city B has a "south" connection to some city "C"
//   instead) then we abort the simulator with an error.

// Additional indices for SNode.roads
const WEST  int = 2;
const NORTH int = 3;

type SNodeArray []SNode         // a city data store

type SNodeMap map[string]int    // index into a city data store (access city struct's index by city name)

type SNode struct {
     cityName     string     // Name of the city ("" is an invalid name)
     roads        [4]int     // Index into a city data store of adjacent cities in the four directions
     sroads       [4]string  // Names of adjacent cities in the four directions (for the first parser pass)
}

// ---------------------------------------------------------------------------------------------------
// Print help
// ---------------------------------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------------------------------
// Map file generator
// ---------------------------------------------------------------------------------------------------

func generate(mapfile string, maxx int, maxy int, cd float64, rd float64) {
     fmt.Printf("Will write mapfile '%s' with dimensions %d x %d, city density %f and road density %f.\n", mapfile, maxx, maxy, cd, rd);

     rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

     wmap := make([][]Node, maxy); 

     // Generate cities first, placing them freely over the world matrix.
     // The city names generated here are boring; it's just a string with the city coordinates.

     for y := 0; y < maxy; y++ {
         row := make([]Node, maxx)
         for x := 0; x < maxx; x++ {
             if (rnd.Float64() <= cd) {
                row[x].cityName = fmt.Sprintf("X%dY%d", x, y);
             } else {
                row[x].cityName = "";
             }
         }
         wmap[y] = row;
     }

     // For every two adjacent cities, consider placing a road to connect them.

     for y := 0; y < maxy; y++ {
         for x := 0; x < maxx; x++ {
             if (wmap[y][x].cityName != "") {

                // Consider creating an EAST road to connect City X,Y to City X+1,Y
                if (x < maxx - 1) && (wmap[y][x+1].cityName != "") {
                      wmap[y][x].roads[EAST] = rnd.Float64() <= rd;
                }

                // Consider creating a SOUTH road to connect City X,Y to City X,Y+1
                if (y < maxy - 1) && (wmap[y+1][x].cityName != "") {
                      wmap[y][x].roads[SOUTH] = rnd.Float64() <= rd;
                }
             }
         }
     }

     // Serialize the generated world model to the text file
     // This serializer is optimized to this generator; it doesn't generate north= and west=
     //   roads. However, the file reader in simulate() understands those if you give it a
     //   file provided by a source that uses them.
     
     file, err := os.Create(mapfile)
     if (err != nil) {
        fmt.Printf("ERROR: Cannot write to output file '%s'.\n", mapfile);
     } else {
        defer file.Close()

        for y := 0; y < maxy; y++ {
          for x := 0; x < maxx; x++ {
            cname := wmap[y][x].cityName
            if (cname != "") {
               s := fmt.Sprintf("%s", cname)
               if (wmap[y][x].roads[EAST]) {
                  s += fmt.Sprintf(" east=%s", wmap[y][x+1].cityName);
               }
               if (wmap[y][x].roads[SOUTH]) {
                  s += fmt.Sprintf(" south=%s", wmap[y+1][x].cityName);
               }
               s += "\n"
               file.WriteString(s)
            }
          }
        }
     }

     fmt.Println("Done.");
}

// ---------------------------------------------------------------------------------------------------
// Map file parser and simulator
// ---------------------------------------------------------------------------------------------------

func simulate(mapfile string, numaliens int) {
     fmt.Printf("Will read mapfile '%s' and simulate it with %d aliens.\n", mapfile, numaliens);

     var nodes SNodeArray = nil
     var nodeMap SNodeMap =  make(map[string]int)

     // ---------------------------------------------------------------------------------------------------
     // Read map file.
     // ---------------------------------------------------------------------------------------------------

     file, err := os.Open(mapfile)
     if (err != nil) {
        fmt.Printf("ERROR: Cannot read from input file '%s'.\n", mapfile);
        return
     }
     defer file.Close()

     scanner := bufio.NewScanner(file)
     for scanner.Scan() {

        // Fetch a new line from the input file to process
        line := scanner.Text()

        // Line is some tokens separated by a space
        items := strings.Split(line, " ")

        // If the line isn't empty, it denotes a new city definition
        if (len(items) >= 1) {

           cityName := items[0];

           // Forbid city redefinition
           _, exists := nodeMap[cityName]
           if (exists) {
              fmt.Printf("ERROR: Duplicate city definition found: '%s'.\n", cityName);
              return 
           }

           // Allocate a new city struct with the city name and dummy road pointers 
           newNode := new(SNode);
           newNode.cityName = cityName;
           newNode.roads    = [4]int   {-1, -1, -1, -1};
           newNode.sroads   = [4]string{"", "", "", ""};

           // Parse all DIRECTION=CITY items from this line and apply them to newNode.sroads
           for i := 1; i < len(items); i++ {
              inners := strings.Split(items[i], "=")
              if (len(inners) != 2) {
                 fmt.Printf("ERROR: Syntax error parsing city connection in line '%s'.\n", line);
                 return
              }

              // **********************************************
              // FIXME: Make a name->int const map instead.
              // **********************************************
              var dir int;
              switch inners[0] {
                 case "east":   dir = EAST;
                 case "south":  dir = SOUTH;
                 case "west":   dir = WEST;
                 case "north":  dir = NORTH;
                 default:
                    fmt.Printf("ERROR: Unknown cardinal direction '%s' in line '%s'.\n", inners[0], line);
                    return
              }

              newNode.sroads[dir] = inners[1];
           }

           // Store the first-pass node data in the node array 
           nodes = append(nodes, *newNode);

           // Update the node map that helps us find a city's index in the node array by its name 
           nodeMap[newNode.cityName] = len(nodes) - 1;           
        }
     }

     if err := scanner.Err(); err != nil {
        fmt.Printf("ERROR: Error encountered while parsing input file '%s'.\n", mapfile);
        return
     }

     // ---------------------------------------------------------------------------------------------------
     // Now we have read all of the cities from the file (we only do one reading pass on the file).
     // Compile SNode.sroads to SNode.roads (convert city names into city node indices).
     // We also check that north/south and east/west connections between adjacent cities are consistent.
     // ---------------------------------------------------------------------------------------------------

     for i := 0; i < len(nodes); i++ {

         
     }

     // ---------------------------------------------------------------------------------------------------
     // Run simulator.
     // ---------------------------------------------------------------------------------------------------

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
