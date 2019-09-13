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

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

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
	index        int        // Own index in the SNodeArray
	cityName     string     // Name of the city ("" is an invalid name)
	roads        [4]int     // Index into a city data store of adjacent cities in the four directions, -1 if none
	sroads       [4]string  // Names of adjacent cities in the four directions (for the first parser pass), "" if none
	dead         bool       // Set to true if the city has been destroyed
	alienid      int        // Alien that is present in this city, or -1 if none
}

type AlienArray []int        // Index is alien number, value is index into a SNodeArray (i.e. which city)

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

	wmap := make([][]Node, maxy);

	// Generate cities first, placing them freely over the world matrix.
	// The city names generated here are boring; it's just a string with the city coordinates.

	for y := 0; y < maxy; y++ {
		row := make([]Node, maxx)
		for x := 0; x < maxx; x++ {
			if (rnd.Float64() <= cd) {
				row[x].cityName = fmt.Sprintf("X%dY%d", x, y)
			} else {
				row[x].cityName = ""
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
					wmap[y][x].roads[EAST] = rnd.Float64() <= rd
				}

				// Consider creating a SOUTH road to connect City X,Y to City X,Y+1
				if (y < maxy - 1) && (wmap[y+1][x].cityName != "") {
					wmap[y][x].roads[SOUTH] = rnd.Float64() <= rd
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
		fmt.Printf("ERROR: Cannot write to output file '%s'.\n", mapfile)
	} else {
		defer file.Close()

		for y := 0; y < maxy; y++ {
			for x := 0; x < maxx; x++ {
            cname := wmap[y][x].cityName
            if (cname != "") {
               s := fmt.Sprintf("%s", cname)
               if (wmap[y][x].roads[EAST]) {
                  s += fmt.Sprintf(" east=%s", wmap[y][x+1].cityName)
               }
               if (wmap[y][x].roads[SOUTH]) {
                  s += fmt.Sprintf(" south=%s", wmap[y+1][x].cityName)
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
	fmt.Printf("Will read mapfile '%s' and simulate it with %d aliens.\n", mapfile, numaliens)

	var nodes SNodeArray = nil
	var nodeMap SNodeMap =  make(map[string]int)

	// ---------------------------------------------------------------------------------------------------
	// Read map file.
	// ---------------------------------------------------------------------------------------------------

	file, err := os.Open(mapfile)
	if (err != nil) {
		fmt.Printf("ERROR: Cannot read from input file '%s'.\n", mapfile)
		return
	}
	defer file.Close()

	// Each new SNode is pushed to the end of the SNodeArray
	var nextIndex = 0;

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
				fmt.Printf("ERROR: Duplicate city definition found: '%s'.\n", cityName)
				return
			}

			// Allocate a new city struct with the city name and dummy road pointers
			newNode := new(SNode);
			newNode.cityName = cityName;
			newNode.index    = nextIndex;
			nextIndex ++;
			newNode.roads    = [4]int   {-1, -1, -1, -1};
			newNode.sroads   = [4]string{"", "", "", ""};
			newNode.dead     = false;
			newNode.alienid  = -1;

			// Parse all DIRECTION=CITY items from this line and apply them to newNode.sroads
			for i := 1; i < len(items); i++ {
				inners := strings.Split(items[i], "=")
				if (len(inners) != 2) {
					fmt.Printf("ERROR: Syntax error parsing city connection in line '%s'.\n", line)
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

				var neighborName = inners[1];
				if (neighborName == cityName) {
					fmt.Printf("ERROR: City '%s' is being defined as a neighbor of itself.\n", cityName)
					return
				}
				newNode.sroads[dir] = neighborName;
			}

			// Store the first-pass node data in the node array
			nodes = append(nodes, *newNode);

			// Update the node map that helps us find a city's index in the node array by its name
			nodeMap[newNode.cityName] = newNode.index;

			// FIXME: change to an assert
			if (len(nodes) - 1 != newNode.index) {
				fmt.Printf("ERROR: The file reader is broken. Expected index %i, found %i.\n", newNode.index, len(nodes) - 1)
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("ERROR: Error encountered while parsing input file '%s'.\n", mapfile)
		return
	}

	// ---------------------------------------------------------------------------------------------------
	// Now we have read all of the cities from the file (we only do one reading pass on the file).
	// Compile SNode.sroads to SNode.roads (convert city names into city node indices).
	// We also check that north/south and east/west connections between adjacent cities are consistent.
	// ---------------------------------------------------------------------------------------------------

	fmt.Printf("Successfully read %d cities from the input file. Checking road links...\n", len(nodes))

	for i := 0; i < len(nodes); i++ {

		var node *SNode = &nodes[i]

		for d := 0; d < 4; d++ {

			neighborName := node.sroads[d];

			if (neighborName == "") {
				continue
			}

			idx, ok := nodeMap[neighborName];
			if (! ok) {
				fmt.Printf("ERROR: City '%s' references an adjacent but non-existing city '%s'.\n", node.cityName, neighborName)
				return
			}

			node.roads[d] = idx;

			// Now, either the neighbor hasn't defined the backlink to us, or if they did, it must point
			//   to us as well. If they did not define it, we will set it now.
			// (We use the node name for this check, since it is filled up from the previous pass)

			// This converds the ESWN "d" into its cardinal opposite as "od"
			// E.g. NORTH ( 3 ), becomes SOUTH ( 1 ).
			od := d;
			od += 2;
			od = od % 4;

			var neighNode *SNode = &nodes[idx];

			if (neighNode.sroads[od] == "") || (neighNode.sroads[od] == node.sroads[d]) {
				neighNode.roads[od] = node.index;
			} else {
				fmt.Printf("ERROR: City '%s' declares a %d road to city '%s', but the inverse %d road points to '%s' instead.\n",
					node.cityName, d, neighNode.cityName, od, neighNode.sroads[od])
				return
			}
		}
	}

	fmt.Println("Done reading input file.");

	// ---------------------------------------------------------------------------------------------------
	// Alien spawn phase.
	// Spawn the aliens randomly, one after the other.
	// If two aliens are spawned in the same city, they die and the city is destroyed.
	// If we run out of cities before all aliens are spawned, the simulation ends and no result file
	//   is written (empty city).
	// ---------------------------------------------------------------------------------------------------

	fmt.Printf("\nSimulation Phase #1: Spawning %d aliens at random cities.\n", numaliens);

	var liveAlienCounter = 0

	var aliens AlienArray = make([]int, numaliens);

	// Initialize all aliens as dead (FIXME: surely there's a better way to do this)

	for i := 0; i < numaliens; i++ {
		aliens[i] = -1
	}

	// Place aliens in sequence.

	for i := 0; i < numaliens; i++ {

		// Choose a random city index to place the next alien.

		chosenCityIndex := -1;
		tryCityIndex := rnd.Intn(len(nodes));

		for cs := 0; cs < len(nodes); cs ++ {

			// Attempt to place alien in the city pointed by the index.
			// If that city was already destroyed, try the next city in the array.

			if (! nodes[tryCityIndex].dead) {
				chosenCityIndex = tryCityIndex
				break
			}

			tryCityIndex ++
			if (tryCityIndex >= len(nodes)) {
				tryCityIndex = 0
			}
		}

		// Check if we have zero cities left.

		if (chosenCityIndex == -1) {
			fmt.Printf("Simulation has ended at Phase #1: no cities left to place Alien #%d. The resulting map is empty (no result map file written).\n", i)
			return
		}

		// Place the alien.

		aliens[i] = chosenCityIndex
		liveAlienCounter ++

		// Check if that alien placement caused a fight.
		// If it did, destroy the city and the two aliens involved.

		existingAlienIdx := nodes[chosenCityIndex].alienid

		if (existingAlienIdx != -1) {

			fmt.Printf("City '%s' has been destroyed by spawning Alien #%d on top of Alien #%d!\n", nodes[chosenCityIndex].cityName, i, existingAlienIdx)

			// Just mark the city as dead
			nodes[chosenCityIndex].dead = true

			// Dead aliens are in no city
			aliens[i] = -1
			aliens[existingAlienIdx] = -1

			liveAlienCounter -= 2
		} else {

			// No fight, so just cache the alien's city location in the city node itself
			nodes[chosenCityIndex].alienid = i;
		}
	}

	// ---------------------------------------------------------------------------------------------------
	// Alien movement phase
	// ---------------------------------------------------------------------------------------------------

	fmt.Println("\nSimulation Phase #2: Moving aliens.\n");

	// We are going to run at most 10,000 movement steps.
	// Each movement step involves moving each alien randomly across a valid road to a city that has
	//   not been destroyed (some aliens can be trapped and unable to move, but if there IS a single
	//   valid path out of their current city, they must be able to take it).

	for r := 0; r < 10000; r++ {

		if (liveAlienCounter <= 0) {
			fmt.Printf("We have %d aliens left alive at iteration %d. Stopping the simulator.\n", liveAlienCounter, r)
			break
		}

		for i := 0; i < numaliens; i++ {

			if (aliens[i] == -1) {
				continue    // skip movement on dead aliens
			}

			// Get a reference to the simulation node where Alien #"i" is

			var anode *SNode = &nodes[aliens[i]]

			// Choose one of the four directions to roam

			chosenDirection := -1;
			destCityIndex   := -1;
			tryDirection    := rnd.Intn(4);

			for dr := 0; dr < 4; dr ++ {

				// Check if that direction is a valid movement direction

				destCityIndex = anode.roads[tryDirection]

				// Skip roads to nowhere (-1) and roads to cities that are already dead
				if (destCityIndex != -1) && (! nodes[destCityIndex].dead) {
					chosenDirection = tryDirection
					break
				}

				tryDirection ++
				if (tryDirection >= 4) { // FIXME: replace all magic "4"s with MAX_DIRECTION
					tryDirection = 0
				}
			}

			// Check if the alien has nowhere to go.

			if (chosenDirection == -1) {
				continue // Alien is just trapped.
			}

			// Move the alien.

			// FIXME: Should be an assert.
			if (destCityIndex == -1) || (nodes[destCityIndex].dead) {
				fmt.Println("ERROR: Simulator has a bug, moving Alien #%d to a bad destCityIndex %d.", i, destCityIndex)
				return
			}

			nodes[aliens[i]].alienid = -1    // remove this alien from the previous location's alienid cache

			aliens[i] = destCityIndex;

			// Check if the destination city (where alien i moved in) didn't already have an alien in it.
			// If so, they fight, both die and the city is destroyed.

			existingAlienIdx := nodes[destCityIndex].alienid

			if (existingAlienIdx != -1) {
				fmt.Printf("City '%s' has been destroyed by Alien #%d and Alien #%d!\n", nodes[destCityIndex].cityName, i, existingAlienIdx)

				// Just mark the city as dead
				nodes[destCityIndex].dead = true

				// Dead aliens are in no city
				aliens[i] = -1
				aliens[existingAlienIdx] = -1

				liveAlienCounter -= 2
			} else {

				// Cache the alien into the new location
				nodes[destCityIndex].alienid = i
			}
		}
	}

	fmt.Printf("Simulation complete. Aliens remaining alive: %d\n", liveAlienCounter);

	// ---------------------------------------------------------------------------------------------------
	// Serialize the simulator data model to "<mapfile>.result"
	// ---------------------------------------------------------------------------------------------------

	resultFileName := mapfile + ".result"

	fmt.Printf("\nWriting resulting map file to '%s'.\n", resultFileName);

	ofile, oerr := os.Create(resultFileName)
	if (oerr != nil) {
		fmt.Printf("ERROR: Cannot write to simulation result output file '%s'.\n", resultFileName)
	} else {
		defer ofile.Close()

		for i := 0; i < len(nodes); i++ {

			// Skip dead cities
			if (nodes[i].dead) {
				continue
			}

			// Line starts with the name of the non-destroyed city
			line := nodes[i].cityName;

			// Then we look for all valid directions that link to other non-dead
			//   cities and append them to the output line
			for d := 0; d < 4; d++ {

				otherIdx := nodes[i].roads[d]

				// No road
				if (otherIdx == -1) {
					continue
				}

				// Leads to dead city
				if (nodes[otherIdx].dead) {
					continue
				}

				// It's good

				otherCityName := nodes[otherIdx].cityName
				directionName := "ERROR"

				// ****************************
				// FIXME: Do it the right way
				// ****************************
				switch d {
				case EAST:  directionName = "east"
				case SOUTH: directionName = "south"
				case WEST:  directionName = "west"
				case NORTH: directionName = "north"
				}

				line += " " + directionName + "=" + otherCityName;
			}

			line += "\n";

			// Write out the line
			ofile.WriteString(line)
		}
	}

	fmt.Println("Done.");
}

// ---------------------------------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------------------------------

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
