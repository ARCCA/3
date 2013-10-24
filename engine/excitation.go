package engine

import (
	"github.com/mumax/3/cuda"
	"github.com/mumax/3/data"
	"github.com/mumax/3/script"
	"log"
	"math"
	"reflect"
)

// An excitation, typically field or current,
// can be defined region-wise plus extra mask*multiplier terms.
type excitation struct {
	perRegion  VectorParam // Region-based excitation
	extraTerms []mulmask   // add extra mask*multiplier terms
}

type mulmask struct {
	mul  func() float64
	mask *data.Slice
}

func (e *excitation) init(name, unit, desc string) {
	e.perRegion.init(name+"_perRegion", unit, "(internal)")
	DeclLValue(name, e, cat(desc, unit))
}

func (e *excitation) addTo(dst *data.Slice) {
	if !e.perRegion.isZero() {
		cuda.RegionAddV(dst, e.perRegion.gpuLUT(), regions.Gpu(), 0)
	}
	for _, t := range e.extraTerms {
		var mul float32 = 1
		if t.mul != nil {
			mul = float32(t.mul())
		}
		cuda.Madd2(dst, dst, t.mask, 1, mul)
	}
}

func (e *excitation) isZero() bool {
	return e.perRegion.isZero() && len(e.extraTerms) == 0
}

func (e *excitation) Slice() (*data.Slice, bool) {
	buf := cuda.Buffer(e.NComp(), e.Mesh())
	cuda.Zero(buf)
	e.addTo(buf)
	return buf, true
}

// Add an extra mask*multiplier term to the excitation.
func (e *excitation) Add(mask *data.Slice, f script.ScalarFunction) {
	var mul func() float64
	if f != nil {
		if f.Cnst() {
			val := f.Float()
			mul = func() float64 {
				return val
			}
		} else {
			mul = func() float64 {
				return f.Float()
			}
		}
	}
	if mask != nil {
		checkNaN(mask, e.Name()+".add()") // TODO: in more places
		mask = data.Resample(mask, e.Mesh().Size())
		mask = assureGPU(mask)
	}
	e.extraTerms = append(e.extraTerms, mulmask{mul, mask})
}

func assureGPU(s *data.Slice) *data.Slice {
	if s.GPUAccess() {
		return s
	} else {
		return cuda.GPUCopy(s)
	}
}

// user script: has to be 3-vector
func (e *excitation) SetRegion(region int, f script.VectorFunction) {
	e.perRegion.SetRegion(region, f)
}

// for gui (nComp agnostic)
func (e *excitation) setRegion(region int, value []float64) {
	log.Println("TODO: time-dep gui")
	e.perRegion.setRegion(region, value)
}

// does not use extramask!
func (e *excitation) getRegion(region int) []float64 {
	return e.perRegion.getRegion(region)
}

func (e *excitation) TableData() []float64 {
	return e.perRegion.getRegion(0)
}

func (p *excitation) Region(r int) TableData {
	return p.perRegion.Region(r)
}

func (e *excitation) IsUniform() bool {
	return e.perRegion.IsUniform()
}

func (e *excitation) SetValue(v interface{}) {
	e.perRegion.SetValue(v) // allows function of time
}

func (e *excitation) Name() string            { return e.perRegion.Name() }
func (e *excitation) Unit() string            { return e.perRegion.Unit() }
func (e *excitation) NComp() int              { return e.perRegion.NComp() }
func (e *excitation) Mesh() *data.Mesh        { return &globalmesh }
func (e *excitation) Eval() interface{}       { return e }
func (e *excitation) Type() reflect.Type      { return reflect.TypeOf(new(excitation)) }
func (e *excitation) InputType() reflect.Type { return script.VectorFunction_t }

func checkNaN(s *data.Slice, name string) {
	h := s.Host()
	for _, h := range h {
		for _, v := range h {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				log.Fatalln("NaN or Inf in", name)
			}
		}
	}
}
