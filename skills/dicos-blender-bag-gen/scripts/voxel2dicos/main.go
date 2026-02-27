// voxel2dicos reads a raw voxel volume and optional threats.json exported
// from Blender and writes DICOS files:
//   - CT Image (.dcs) — multi-frame volume, one axial slice per frame
//   - TDR (.dcs) — Threat Detection Report with PTO bounding boxes (if threats present)
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"gitlab.ses.psdo.leidos.com/enterprise-security-platform/prosight-devices/dicos.go/pkg/dicos"
)

type rawHeader struct {
	Width, Height, Depth         uint32
	SpacingX, SpacingY, SpacingZ float64
	OriginX, OriginY, OriginZ   float64
}

type threatFile struct {
	Threats []threatEntry `json:"threats"`
}

type threatEntry struct {
	Label      string  `json:"label"`
	Category   string  `json:"category"`
	Flag       string  `json:"flag"`
	Probability float64 `json:"probability"`
	BBoxMM     struct {
		Min [3]float64 `json:"min"`
		Max [3]float64 `json:"max"`
	} `json:"bbox_mm"`
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

	// Derive threats.json path from raw path directory
	threatsPath := filepath.Join(filepath.Dir(rawPath), "threats.json")

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

	ct.Patient.SetPatientName("BlenderSim", "Screening", "", "", "")
	ct.Patient.PatientID = "BAG-CT-001"
	ct.Series.Modality = "CT"
	ct.Series.SeriesDescription = "Simulated CT scan — airport screening tray"
	ct.Series.SeriesNumber = 1
	ct.Study.StudyDescription = "Airport Security Screening Simulation"
	ct.Equipment.Manufacturer = "dicos.go Blender Voxelizer"
	ct.Equipment.StationName = "BLENDER-SIM"
	ct.Equipment.InstitutionName = "DICOS.go Project"

	ct.CTImageMod.KVP = 140
	ct.CTImageMod.ExposureTime = 500
	ct.CTImageMod.XRayTubeCurrent = 300
	ct.CTImageMod.FilterType = "BODY"
	ct.CTImageMod.ConvolutionKernel = "STANDARD"
	ct.CTImageMod.AcquisitionType = "SPIRAL"
	ct.CTImageMod.DataCollectionDiameter = 620
	ct.CTImageMod.ReconstructionDiameter = 640
	ct.CTImageMod.ImageType = []string{"ORIGINAL", "PRIMARY", "AXIAL"}

	ct.RescaleIntercept = -1024.0
	ct.RescaleSlope = 1.0
	ct.RescaleType = "HU"

	ct.CTImageMod.WindowCenter = 1500
	ct.CTImageMod.WindowWidth = 4000

	ct.ImagePlane.PixelSpacing = [2]float64{hdr.SpacingY, hdr.SpacingX}
	ct.ImagePlane.SliceThickness = hdr.SpacingZ
	ct.ImagePlane.SpacingBetweenSlices = hdr.SpacingZ
	ct.ImagePlane.ImageOrientationPatient = [6]float64{1, 0, 0, 0, 1, 0}
	ct.ImagePlane.ImagePositionPatient = [3]float64{hdr.OriginX, hdr.OriginY, hdr.OriginZ}

	ct.FrameOfReference.FrameOfReferenceUID = dicos.GenerateUID("1.2.826.0.1.3680043.8.498.")
	ct.FrameOfReference.PositionReferenceIndicator = "BB"

	ct.Rows = height
	ct.Columns = width
	ct.BitsAllocated = 16
	ct.BitsStored = 16
	ct.HighBit = 15
	ct.PixelRepresent = 0
	ct.SamplesPerPixel = 1
	ct.PhotometricInterp = "MONOCHROME2"

	ct.SetPixelData(height, width, voxels)

	n, err := ct.Write(outPath)
	if err != nil {
		log.Fatalf("write DICOS CT: %v", err)
	}

	fmt.Printf("\nDICOS CT written: %s (%d bytes, %.1f MB)\n", outPath, n, float64(n)/1024/1024)
	fmt.Printf("  %d frames, %dx%d per frame\n", depth, width, height)
	fmt.Printf("  Spacing: %.2f x %.2f x %.2f mm\n", hdr.SpacingX, hdr.SpacingY, hdr.SpacingZ)

	// --- Read threats and build TDR ---
	threatData, err := os.ReadFile(threatsPath)
	if err != nil {
		fmt.Printf("\nNo threats.json found — skipping TDR\n")
		return
	}

	var tf threatFile
	if err := json.Unmarshal(threatData, &tf); err != nil {
		log.Fatalf("parse threats.json: %v", err)
	}

	if len(tf.Threats) == 0 {
		fmt.Printf("\nNo threats in threats.json — skipping TDR\n")
		return
	}

	tdr := dicos.NewThreatDetectionReport()
	tdr.AlarmDecision = "ALARM"
	tdr.Series.Modality = "TDR"
	tdr.Equipment.Manufacturer = "dicos.go Blender Voxelizer"
	tdr.ReferencedSOPClassUID = "1.2.840.10008.5.1.4.1.1.2" // CT Image Storage
	tdr.ReferencedSOPInstanceUID = ct.SOPCommon.SOPInstanceUID

	for i, t := range tf.Threats {
		// Convert absolute mm to volume-relative mm
		bbMin := [3]float32{
			float32(t.BBoxMM.Min[0] - hdr.OriginX),
			float32(t.BBoxMM.Min[1] - hdr.OriginY),
			float32(t.BBoxMM.Min[2] - hdr.OriginZ),
		}
		bbMax := [3]float32{
			float32(t.BBoxMM.Max[0] - hdr.OriginX),
			float32(t.BBoxMM.Max[1] - hdr.OriginY),
			float32(t.BBoxMM.Max[2] - hdr.OriginZ),
		}

		tdr.PTOs = append(tdr.PTOs, dicos.PotentialThreatObject{
			ID:          i + 1,
			Label:       t.Label,
			OOIType:     t.Category,
			Probability: float32(t.Probability),
			Confidence:  float32(t.Probability),
			BoundingBox: &dicos.BoundingBox{
				TopLeft:     bbMin,
				BottomRight: bbMax,
			},
		})

		fmt.Printf("\n  PTO %d: %s [%s] prob=%.2f\n", i+1, t.Label, t.Category, t.Probability)
		fmt.Printf("    BBox: [%.1f,%.1f,%.1f] - [%.1f,%.1f,%.1f] mm\n",
			bbMin[0], bbMin[1], bbMin[2], bbMax[0], bbMax[1], bbMax[2])
	}

	tdrPath := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + "_tdr.dcs"
	tn, err := tdr.Write(tdrPath)
	if err != nil {
		log.Fatalf("write DICOS TDR: %v", err)
	}

	fmt.Printf("\nDICOS TDR written: %s (%d bytes)\n", tdrPath, tn)
	fmt.Printf("  Alarm Decision: ALARM\n")
	fmt.Printf("  %d PTOs\n", len(tdr.PTOs))
	fmt.Printf("  References CT: %s\n", ct.SOPCommon.SOPInstanceUID)
}
