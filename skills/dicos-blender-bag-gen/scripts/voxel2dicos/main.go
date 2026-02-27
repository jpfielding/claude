// voxel2dicos reads a raw voxel volume exported from Blender and writes
// a multi-frame DICOS CT Image file. The volume is stored as axial slices
// (one frame per Z-level), preserving the three standard CT viewing planes:
//   - Axial   (XY plane, one per frame)
//   - Coronal (XZ plane, reconstructed from frames)
//   - Sagittal (YZ plane, reconstructed from frames)
//
// Usage:
//
//	go run . [raw_input] [dcs_output]
//	go run . tmp/voxels.raw tmp/bag_ct.dcs
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/jpfielding/dicos.go/pkg/dicos"
)

// rawHeader matches the binary layout written by the Blender voxelizer.
type rawHeader struct {
	Width, Height, Depth         uint32
	SpacingX, SpacingY, SpacingZ float64
	OriginX, OriginY, OriginZ   float64
}

func main() {
	rawPath := "tmp/voxels.raw"
	outPath := "tmp/bag_ct.dcs"

	if len(os.Args) > 1 {
		rawPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		outPath = os.Args[2]
	}

	// --- Read raw volume ---
	f, err := os.Open(rawPath)
	if err != nil {
		log.Fatalf("open raw: %v", err)
	}
	defer f.Close()

	var hdr rawHeader
	if err := binary.Read(f, binary.LittleEndian, &hdr); err != nil {
		log.Fatalf("read header: %v", err)
	}

	width := int(hdr.Width)
	height := int(hdr.Height)
	depth := int(hdr.Depth)
	totalVoxels := width * height * depth

	fmt.Printf("Volume: %dx%dx%d (%d voxels)\n", width, height, depth, totalVoxels)
	fmt.Printf("Spacing: %.2f x %.2f x %.2f mm\n", hdr.SpacingX, hdr.SpacingY, hdr.SpacingZ)
	fmt.Printf("Origin:  %.1f, %.1f, %.1f mm\n", hdr.OriginX, hdr.OriginY, hdr.OriginZ)

	voxels := make([]uint16, totalVoxels)
	if err := binary.Read(f, binary.LittleEndian, voxels); err != nil {
		log.Fatalf("read voxels: %v", err)
	}

	// Compute min/max for window/level
	var vmin, vmax uint16 = math.MaxUint16, 0
	var nonzero int
	for _, v := range voxels {
		if v > 0 {
			nonzero++
			if v < vmin {
				vmin = v
			}
			if v > vmax {
				vmax = v
			}
		}
	}
	fmt.Printf("Non-zero: %d (%.1f%%), range: [%d, %d]\n", nonzero, 100*float64(nonzero)/float64(totalVoxels), vmin, vmax)

	// --- Build DICOS CT Image ---
	ct := dicos.NewCTImage()

	// Patient / Study / Series metadata
	ct.Patient.SetPatientName("BlenderSim", "CarryOn", "", "", "")
	ct.Patient.PatientID = "BAG-CT-001"
	ct.Series.Modality = "CT"
	ct.Series.SeriesDescription = "Simulated CT scan of carry-on bag in screening tray"
	ct.Series.SeriesNumber = 1
	ct.Study.StudyDescription = "Airport Security Screening Simulation"
	ct.Equipment.Manufacturer = "dicos.go Blender Voxelizer"
	ct.Equipment.StationName = "BLENDER-SIM"
	ct.Equipment.InstitutionName = "DICOS.go Project"

	// CT Image parameters (simulated Smiths HI-SCAN 6040 CTiX)
	ct.CTImageMod.KVP = 140                   // Dual-energy CT typically 80/140 kV
	ct.CTImageMod.ExposureTime = 500           // ms
	ct.CTImageMod.XRayTubeCurrent = 300        // mA
	ct.CTImageMod.FilterType = "BODY"
	ct.CTImageMod.ConvolutionKernel = "STANDARD"
	ct.CTImageMod.AcquisitionType = "SPIRAL"
	ct.CTImageMod.DataCollectionDiameter = 620 // Scanner tunnel width mm
	ct.CTImageMod.ReconstructionDiameter = 640 // Matches tray width
	ct.CTImageMod.ImageType = []string{"ORIGINAL", "PRIMARY", "AXIAL"}

	// Rescale: stored values map to approximate Hounsfield-like units
	// RescaleIntercept = -1024, so air (stored 0) → HU -1024
	// Water stored ~1024 → HU 0
	ct.RescaleIntercept = -1024.0
	ct.RescaleSlope = 1.0
	ct.RescaleType = "HU"

	// Window/Level presets for viewing
	ct.CTImageMod.WindowCenter = 2000
	ct.CTImageMod.WindowWidth = 6000

	// Image Plane
	ct.ImagePlane.PixelSpacing = [2]float64{hdr.SpacingY, hdr.SpacingX} // row\col
	ct.ImagePlane.SliceThickness = hdr.SpacingZ
	ct.ImagePlane.SpacingBetweenSlices = hdr.SpacingZ
	ct.ImagePlane.ImageOrientationPatient = [6]float64{1, 0, 0, 0, 1, 0} // Standard axial
	ct.ImagePlane.ImagePositionPatient = [3]float64{hdr.OriginX, hdr.OriginY, hdr.OriginZ}

	// Frame of Reference
	ct.FrameOfReference.FrameOfReferenceUID = dicos.GenerateUID("1.2.826.0.1.3680043.8.498.")
	ct.FrameOfReference.PositionReferenceIndicator = "BB" // Bounding box reference

	// Image dimensions
	ct.Rows = height
	ct.Columns = width
	ct.BitsAllocated = 16
	ct.BitsStored = 16
	ct.HighBit = 15
	ct.PixelRepresent = 0 // unsigned
	ct.SamplesPerPixel = 1
	ct.PhotometricInterp = "MONOCHROME2"

	// Set pixel data: all Z slices as frames (axial view is native)
	ct.SetPixelData(height, width, voxels)

	// --- Write DICOS file ---
	n, err := ct.Write(outPath)
	if err != nil {
		log.Fatalf("write DICOS: %v", err)
	}

	fmt.Printf("\nDICOS CT written: %s (%d bytes, %.1f MB)\n", outPath, n, float64(n)/1024/1024)
	fmt.Printf("  %d frames (axial slices), %dx%d per frame\n", depth, width, height)
	fmt.Printf("  Pixel spacing: %.2f x %.2f mm\n", hdr.SpacingX, hdr.SpacingY)
	fmt.Printf("  Slice spacing: %.2f mm\n", hdr.SpacingZ)
	fmt.Printf("  Transfer syntax: Explicit VR Little Endian (uncompressed)\n")
	fmt.Printf("  Modality: CT\n")
	fmt.Printf("\nViewing planes available from this volume:\n")
	fmt.Printf("  Axial   (XY): %d slices at %dx%d\n", depth, width, height)
	fmt.Printf("  Coronal (XZ): %d slices at %dx%d\n", height, width, depth)
	fmt.Printf("  Sagittal(YZ): %d slices at %dx%d\n", width, height, depth)
}
