package synthesis

import (
	"byrd/internal/common"
	"fmt"
	"math"
)

type PolyphaseState struct {
	v [1024]float64
}

var synthesisMatrix = buildSynthesisMatrix()

func buildSynthesisMatrix() [64][32]float64 {
	var m [64][32]float64
	for i := range 64 {
		for k := range 32 {
			m[i][k] = math.Cos(math.Pi / 64 * float64((i+16)*(2*k+1)))
		}
	}
	return m
}

func SynthesizeSubbandSamples(in []float64, state *PolyphaseState, out []float64) error {
	if len(in) != 32 {
		return fmt.Errorf("polyphase synthesis requires 32 subband samples: got %d", len(in))
	}
	if state == nil {
		return fmt.Errorf("nil polyphase state")
	}
	if len(out) != 32 {
		return fmt.Errorf("polyphase synthesis requires 32 output samples: got %d", len(out))
	}

	var x [64]float64
	for i := range 64 {
		sum := 0.0
		for k := range 32 {
			sum += in[k] * synthesisMatrix[i][k]
		}
		x[i] = sum
	}

	copy(state.v[64:], state.v[:960])
	copy(state.v[:64], x[:])

	var u [512]float64
	for i := 0; i < 8; i++ {
		copy(u[i*64:i*64+32], state.v[i*128:i*128+32])
		copy(u[i*64+32:i*64+64], state.v[i*128+96:i*128+128])
	}

	for j := range u {
		u[j] *= common.SynthDtbl[j]
	}

	for j := range 32 {
		sum := 0.0
		for i := 0; i < 16; i++ {
			sum += u[j+32*i]
		}
		out[j] = sum
	}

	return nil
}

func SynthesizeGranule(in *[32][18]float64, state *PolyphaseState, out *[576]float64) error {
	if in == nil {
		return fmt.Errorf("nil hybrid input")
	}
	if state == nil {
		return fmt.Errorf("nil polyphase state")
	}
	if out == nil {
		return fmt.Errorf("nil pcm output")
	}

	var subbandIn [32]float64
	var slotOut [32]float64
	for ss := 0; ss < 18; ss++ {
		for sb := 0; sb < 32; sb++ {
			subbandIn[sb] = in[sb][ss]
		}
		if err := SynthesizeSubbandSamples(subbandIn[:], state, slotOut[:]); err != nil {
			return err
		}
		copy(out[ss*32:(ss+1)*32], slotOut[:])
	}

	return nil
}
