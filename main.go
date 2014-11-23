package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/phil-mansfield/gotetra/scripts/helper"
	"github.com/phil-mansfield/gotetra/density"
	"github.com/phil-mansfield/gotetra/geom"
)

const (
	vecBufLen = 1<<10
)

func main() {
	var (
		x, y, z int
		createCatalog, compareDensity bool
		cells int
	)

	outPath := flag.String("Log", "", "Location to write log statements to. " + 
		"Default is stderr.")

	flag.IntVar(&x, "X", -1, "z location of operation.")
	flag.IntVar(&y, "Y", -1, "y location of operation.")
	flag.IntVar(&z, "Z", -1, "z location of operation.")
	flag.IntVar(&cells, "Cells", -1, "Width of grid in cells.")

	flag.BoolVar(&createCatalog, "CreateCatalog", false,
		"Generate gotetra catalogs from gadget catalogs.")
	flag.BoolVar(&compareDensity, "CompareDensity", false,
		"Compare different methods of calculating densities.")

	flag.Parse()

	if *outPath == "" {
		log.SetOutput(os.Stderr)
	} else {
		if lf, err := os.Create(*outPath); err != nil {
			log.Fatalln(err.Error())
		} else {
			log.SetOutput(lf)
			defer lf.Close()
		}
	}

	modeName := checkMode(createCatalog, compareDensity)

	switch {
	case createCatalog:
		checkCells(cells, modeName)

		args := flag.Args()
		if len(args) <= 1 {
			log.Fatalf(
				"Mode %s requires source files and a target directory.",
				modeName,
			)
		}

		sources := args[1: len(args) - 1]
		targetDir := args[len(args) - 1]

		createCatalogsMain(x, y, z, cells, sources, targetDir)
		
	case compareDensity:
		checkCells(cells, modeName)
		checkCoords(x, y, x, modeName)

		args := flag.Args()
		if len(args) != 1 {
			log.Fatalf("Mode %s requires target directory.", modeName)
		}

		source := args[len(args) - 1]
		compareDensityMain(x, y, z, cells, source)
	}
}

func checkCells(cells int, modeName string) {
	if cells == -1 {
		log.Fatalf(
			"The mode %s requires a positive number of cells.\n", modeName,
		)
	}
}

func checkCoords(x, y, z int, modeName string) {
	if x == -1 {
		log.Fatalf("The mode %s requires an x location.\n", modeName)
	} else if y == -1 {
		log.Fatalf("The mode %s requires a y location.\n", modeName)
	} else if z == -1 {
		log.Fatalf("The mode %s requires a z location.\n", modeName)
	}
}

func checkMode(createCatalog, compareDensity bool) string {
	n := 0
	modeStr := ""

	if createCatalog {
		n++
		modeStr = "CreateCatalog"
	}

	if compareDensity {
		n++
		modeStr = "CompareDensity"
	}

	if n != 1 {
		log.Fatalf("Given %d mode flags, but exactly 1 is required.\n", n)
	}

	return modeStr
}

func createCatalogsMain(x, y, z, cells int, sources []string, target string) {
	log.Fatalf("CreateCatalogs not yet [re]implemented.")
}

func compareDensityMain(x, y, z, cells int, sourceDir string) {
	log.Printf("Starting to read catalogs at (%d, %d, %d)\n", x, y, z)
	h0, man, centerPs := helper.ReadCatalogs(sourceDir, x, y, z, 1)
	log.Println("Finished reading catalogs.")

	g, bg := density.Bounds(cells, int(h0.GridWidth), x, y, z)

	ngpRhos := make([]float64, cells * cells * cells)
	ngpIntr := density.NewInterpolator(
		density.NearestGridPoint, g, bg, h0.Width, h0.CountWidth, man, ngpRhos,
	)

	cicRhos := make([]float64, cells * cells * cells)
	cicIntr := density.NewInterpolator(
		density.CloudInCell, g, bg, h0.Width, h0.CountWidth, man, cicRhos,
	)

	mcRhos10 := make([]float64, cells * cells * cells)
	mcIntr10 := density.NewMonteCarloInterpolator(
		g, bg, h0.Width, h0.CountWidth, man, 10, mcRhos10,
	)

	mcRhos100 := make([]float64, cells * cells * cells)
	mcIntr100 := density.NewMonteCarloInterpolator(
		g, bg, h0.Width, h0.CountWidth, man, 100, mcRhos100,
	)

	//mcRhos1000 := make([]float64, cells * cells * cells)
	//mcIntr1000 := density.NewMonteCarloInterpolator(
	//	g, bg, h0.Width, h0.CountWidth, man, 1000, mcRhos1000,
	//)

	xsBuf := make([]geom.Vec, vecBufLen)
	idsBuf := make([]int64, vecBufLen)
		
	log.Println("Set up interpolators and buffers.")
	checkLen := len(centerPs) / 20

	bufIdx := 0
	for i := range centerPs {
		if i % checkLen == 0 {
			log.Printf("%d Particles interpolated.\n", i)
		}

		xsBuf[bufIdx] = centerPs[i].Xs
		idsBuf[bufIdx] = centerPs[i].Id

		if bufIdx == len(xsBuf) - 1 {
			ngpIntr.Interpolate(h0.Mass, idsBuf, xsBuf)
			cicIntr.Interpolate(h0.Mass, idsBuf, xsBuf)
			mcIntr10.Interpolate(h0.Mass, idsBuf, xsBuf)
			mcIntr100.Interpolate(h0.Mass, idsBuf, xsBuf)
			// mcIntr1000.Interpolate(h0.Mass, idsBuf, xsBuf)
			bufIdx = 0
		} else {
			bufIdx++
		}
	}
	ngpIntr.Interpolate(h0.Mass, idsBuf[0: bufIdx], xsBuf[0: bufIdx])
	cicIntr.Interpolate(h0.Mass, idsBuf[0: bufIdx], xsBuf[0: bufIdx])
	mcIntr10.Interpolate(h0.Mass, idsBuf[0: bufIdx], xsBuf[0: bufIdx])
	mcIntr100.Interpolate(h0.Mass, idsBuf[0: bufIdx], xsBuf[0: bufIdx])
	// mcIntr1000.Interpolate(h0.Mass, idsBuf[0: bufIdx], xsBuf[0: bufIdx])

	log.Println("Finished interpolation.")

	fmt.Printf("# %12s %12s\n", "NGP", "CIC")
	for i := range ngpRhos {
		fmt.Printf(
			"  %10.5g %10.5g %10.5g %10.5g\n",
			ngpRhos[i], cicRhos[i], mcRhos10[i], mcRhos100[i],
		)
	}

	log.Println("Finished printing density grid.")
}