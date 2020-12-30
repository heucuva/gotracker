package playback

import (
	"time"

	"github.com/gotracker/gomixing/mixing"
	"github.com/gotracker/gomixing/panning"
	"github.com/gotracker/gomixing/volume"
	device "github.com/gotracker/gosound"
	"github.com/gotracker/opl2"

	"gotracker/internal/format/xm/playback/effect"
	"gotracker/internal/format/xm/playback/util"
	"gotracker/internal/player/intf"
	"gotracker/internal/player/render"
)

// RenderOneRow renders the next single row from the song pattern data into a RowRender object
func (m *Manager) renderOneRow() (*device.PremixData, error) {
	preMixRowTxn := m.pattern.StartTransaction()
	postMixRowTxn := m.pattern.StartTransaction()
	defer func() {
		preMixRowTxn.Cancel()
		m.preMixRowTxn = nil
		postMixRowTxn.Cancel()
		m.postMixRowTxn = nil
	}()
	m.preMixRowTxn = preMixRowTxn
	m.postMixRowTxn = postMixRowTxn

	if err := m.startNextRow(); err != nil {
		return nil, err
	}

	preMixRowTxn.Commit()

	finalData := &render.RowRender{}
	premix := &device.PremixData{
		Userdata: finalData,
	}

	m.soundRenderRow(premix)

	finalData.Order = int(m.pattern.GetCurrentOrder())
	finalData.Row = int(m.pattern.GetCurrentRow())
	finalData.RowText = m.getRowText()

	postMixRowTxn.AdvanceRow()

	postMixRowTxn.Commit()
	return premix, nil
}

func (m *Manager) startNextRow() error {
	patIdx, err := m.pattern.GetCurrentPatternIdx()
	if err != nil {
		return err
	}

	pat := m.song.GetPattern(patIdx)
	if pat == nil {
		return intf.ErrStopSong
	}

	rows := pat.GetRows()

	myCurrentRow := m.pattern.GetCurrentRow()

	row := rows.GetRow(myCurrentRow)
	for channelNum, cdata := range row.GetChannels() {
		if channelNum >= m.GetNumChannels() {
			continue
		}

		cs := &m.channels[channelNum]

		cs.ProcessRow(row, cdata, m.globalVolume, m.song, util.CalcSemitonePeriod, m.processEffect)

		cs.ActiveEffect = effect.Factory(cs.GetMemory(), cs.Cmd)
		if cs.ActiveEffect != nil {
			cs.ActiveEffect.PreStart(cs, m)
		}
	}

	return nil
}

func (m *Manager) soundRenderRow(premix *device.PremixData) {
	mix := m.s.Mixer()

	samplerSpeed := m.s.GetSamplerSpeed()
	tickDuration := time.Duration(2500) * time.Millisecond / time.Duration(m.pattern.GetTempo())
	samplesPerTick := int(tickDuration.Seconds() * float64(m.s.SampleRate))

	ticksThisRow := m.pattern.GetTicksThisRow()

	samplesThisRow := int(ticksThisRow) * samplesPerTick

	panmixer := m.s.GetPanMixer()

	centerPanning := panmixer.GetMixingMatrix(panning.CenterAhead)

	premix.SamplesLen = samplesThisRow

	chRrs := make([][]mixing.Data, len(m.channels))
	for ch := range m.channels {
		chRrs[ch] = make([]mixing.Data, ticksThisRow)
	}

	firstOplCh := -1
	for tick := 0; tick < ticksThisRow; tick++ {
		var lastTick = (tick+1 == ticksThisRow)

		for ch := range m.channels {
			cs := &m.channels[ch]
			if m.song.IsChannelEnabled(ch) {
				rr := chRrs[ch]
				cs.RenderRowTick(tick, lastTick, rr, ch, ticksThisRow, mix, panmixer, samplerSpeed, samplesPerTick, centerPanning, tickDuration)

				for len(premix.Data) <= ch {
					premix.Data = append(premix.Data, nil)
				}
				premix.Data[ch] = rr
			}
		}
		if m.opl2 != nil {
			ch := firstOplCh
			for len(premix.Data) <= ch {
				premix.Data = append(premix.Data, nil)
			}
			rr := chRrs[ch]
			m.renderOPL2RowTick(tick, rr, ticksThisRow, mix, panmixer, samplerSpeed, samplesPerTick, centerPanning, tickDuration)
			premix.Data[ch] = rr
		}
	}

	premix.MixerVolume = m.mixerVolume
	if m.opl2 != nil {
		// make room in the mixer for the OPL2 data
		// effectively, we can do this by calculating the new number (+1) of channels from the mixer volume (channels = reciprocal of mixer volume):
		//   numChannels = (1/mv) + 1
		// then by taking the reciprocal of it:
		//   1 / numChannels
		// but that ends up being simplified to:
		//   mv / (mv + 1)
		// and we get protection from div/0 in the process - provided, of course, that the mixerVolume is not exactly -1...
		premix.MixerVolume = m.mixerVolume / (m.mixerVolume + 1)
	}
}

func (m *Manager) renderOPL2RowTick(tick int, mixerData []mixing.Data, ticksThisRow int, mix *mixing.Mixer, panmixer mixing.PanMixer, samplerSpeed float32, tickSamples int, centerPanning volume.Matrix, tickDuration time.Duration) {
	// make a stand-alone data buffer for this channel for this tick
	data := mix.NewMixBuffer(tickSamples)

	opl2data := make([]int32, tickSamples)

	m.opl2.GenerateBlock2(uint(tickSamples), opl2data)

	for i, s := range opl2data {
		sv := volume.Volume(s) / 32768.0
		for c := range data {
			data[c][i] = sv
		}
	}
	mixerData[tick] = mixing.Data{
		Data:       data,
		Pan:        panning.CenterAhead,
		Volume:     util.DefaultVolume * m.globalVolume,
		SamplesLen: tickSamples,
	}
}

func (m *Manager) setOPL2Chip(rate uint32) {
	m.opl2 = opl2.NewChip(rate, false)
	m.opl2.WriteReg(0x01, 0x20) // enable all waveforms
	m.opl2.WriteReg(0x04, 0x00) // clear timer flags
	m.opl2.WriteReg(0x08, 0x40) // clear CSW and set NOTE-SEL
	m.opl2.WriteReg(0xBD, 0x00) // set default notes
}