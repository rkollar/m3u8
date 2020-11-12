package m3u8

/*
 Part of M3U8 parser & generator library.
 This file defines functions related to playlist generation.

 Copyright 2013-2017 The Project Developers.
 See the AUTHORS and LICENSE files at the top-level directory of this distribution
 and at https://github.com/grafov/m3u8/

 ॐ तारे तुत्तारे तुरे स्व
*/

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	ErrPlaylistFull = errors.New("playlist is full")
)

// Set version of the playlist accordingly with section 7
func version(ver *uint8, newver uint8) {
	if *ver < newver {
		*ver = newver
	}
}

func strver(ver uint8) string {
	return strconv.FormatUint(uint64(ver), 10)
}

// Create new empty master playlist.
// Master playlist consists of variants.
func NewMasterPlaylist() *MasterPlaylist {
	p := new(MasterPlaylist)
	p.ver = minver
	return p
}

// Append variant to master playlist.
// This operation does reset playlist cache.
func (p *MasterPlaylist) Append(uri string, chunklist *MediaPlaylist, params VariantParams) {
	v := new(Variant)
	v.URI = uri
	v.Chunklist = chunklist
	v.VariantParams = params
	p.Variants = append(p.Variants, v)
	if len(v.Alternatives) > 0 {
		// From section 7:
		// The EXT-X-MEDIA tag and the AUDIO, VIDEO and SUBTITLES attributes of
		// the EXT-X-STREAM-INF tag are backward compatible to protocol version
		// 1, but playback on older clients may not be desirable.  A server MAY
		// consider indicating a EXT-X-VERSION of 4 or higher in the Master
		// Playlist but is not required to do so.
		version(&p.ver, 4) // so it is optional and in theory may be set to ver.1
		// but more tests required
	}
}

// Generate output in M3U8 format.
func (p *MasterPlaylist) Encode() (buf *bytes.Buffer) {
	buf = bytes.NewBuffer(nil)

	buf.WriteString("#EXTM3U\n#EXT-X-VERSION:")
	buf.WriteString(strver(p.ver))
	buf.WriteRune('\n')

	if p.IndependentSegments() {
		buf.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	}

	// Write any custom master tags
	if p.Custom != nil {
		for _, v := range p.Custom {
			if customBuf := v.Encode(); customBuf != nil {
				buf.WriteString(customBuf.String())
				buf.WriteRune('\n')
			}
		}
	}

	var altsWritten map[string]bool = make(map[string]bool)

	for _, pl := range p.Variants {
		if pl.Alternatives != nil {
			for _, alt := range pl.Alternatives {
				// Make sure that we only write out an alternative once
				altKey := fmt.Sprintf("%s-%s-%s-%s", alt.Type, alt.GroupId, alt.Name, alt.Language)
				if altsWritten[altKey] {
					continue
				}
				altsWritten[altKey] = true

				buf.WriteString("#EXT-X-MEDIA:")
				if alt.Type != "" {
					buf.WriteString("TYPE=") // Type should not be quoted
					buf.WriteString(alt.Type)
				}
				if alt.GroupId != "" {
					buf.WriteString(",GROUP-ID=\"")
					buf.WriteString(alt.GroupId)
					buf.WriteRune('"')
				}
				if alt.Name != "" {
					buf.WriteString(",NAME=\"")
					buf.WriteString(alt.Name)
					buf.WriteRune('"')
				}
				buf.WriteString(",DEFAULT=")
				if alt.Default {
					buf.WriteString("YES")
				} else {
					buf.WriteString("NO")
				}
				if alt.Autoselect != "" {
					buf.WriteString(",AUTOSELECT=")
					buf.WriteString(alt.Autoselect)
				}
				if alt.Language != "" {
					buf.WriteString(",LANGUAGE=\"")
					buf.WriteString(alt.Language)
					buf.WriteRune('"')
				}
				if alt.Forced != "" {
					buf.WriteString(",FORCED=")
					buf.WriteString(alt.Forced)
				}
				if alt.Type == "CLOSED-CAPTIONS" && alt.InstreamID != "" {
					buf.WriteString(",INSTREAM-ID=\"")
					buf.WriteString(alt.InstreamID)
					buf.WriteRune('"')
				}
				if alt.Characteristics != "" {
					buf.WriteString(",CHARACTERISTICS=\"")
					buf.WriteString(alt.Characteristics)
					buf.WriteRune('"')
				}
				if alt.Channels != "" {
					buf.WriteString(",CHANNELS=\"")
					buf.WriteString(alt.Channels)
					buf.WriteRune('"')
				}
				if alt.Subtitles != "" {
					buf.WriteString(",SUBTITLES=\"")
					buf.WriteString(alt.Subtitles)
					buf.WriteRune('"')
				}
				if alt.URI != "" {
					buf.WriteString(",URI=\"")
					buf.WriteString(alt.URI)
					buf.WriteRune('"')
				}
				buf.WriteRune('\n')
			}
		}
		if pl.Iframe {
			buf.WriteString("#EXT-X-I-FRAME-STREAM-INF:")

			buf.WriteString("BANDWIDTH=")
			buf.WriteString(strconv.FormatUint(uint64(pl.Bandwidth), 10))
			if p.ver < 6 {
				buf.WriteString(",PROGRAM-ID=")
				buf.WriteString(strconv.FormatUint(uint64(pl.ProgramId), 10))
			}
			if pl.AverageBandwidth != 0 {
				buf.WriteString(",AVERAGE-BANDWIDTH=")
				buf.WriteString(strconv.FormatUint(uint64(pl.AverageBandwidth), 10))
			}
			if pl.Codecs != "" {
				buf.WriteString(",CODECS=\"")
				buf.WriteString(pl.Codecs)
				buf.WriteRune('"')
			}
			if pl.Resolution != "" {
				buf.WriteString(",RESOLUTION=") // Resolution should not be quoted
				buf.WriteString(pl.Resolution)
			}
			if pl.Video != "" {
				buf.WriteString(",VIDEO=\"")
				buf.WriteString(pl.Video)
				buf.WriteRune('"')
			}
			if pl.VideoRange != "" {
				buf.WriteString(",VIDEO-RANGE=")
				buf.WriteString(pl.VideoRange)
			}
			if pl.HDCPLevel != "" {
				buf.WriteString(",HDCP-LEVEL=")
				buf.WriteString(pl.HDCPLevel)
			}
			if pl.URI != "" {
				buf.WriteString(",URI=\"")
				buf.WriteString(pl.URI)
				buf.WriteRune('"')
			}
			buf.WriteRune('\n')
		} else {
			buf.WriteString("#EXT-X-STREAM-INF:")

			buf.WriteString("BANDWIDTH=")
			buf.WriteString(strconv.FormatUint(uint64(pl.Bandwidth), 10))
			if p.ver < 6 {
				buf.WriteString(",PROGRAM-ID=")
				buf.WriteString(strconv.FormatUint(uint64(pl.ProgramId), 10))
			}
			if pl.AverageBandwidth != 0 {
				buf.WriteString(",AVERAGE-BANDWIDTH=")
				buf.WriteString(strconv.FormatUint(uint64(pl.AverageBandwidth), 10))
			}
			if pl.Codecs != "" {
				buf.WriteString(",CODECS=\"")
				buf.WriteString(pl.Codecs)
				buf.WriteRune('"')
			}
			if pl.Resolution != "" {
				buf.WriteString(",RESOLUTION=") // Resolution should not be quoted
				buf.WriteString(pl.Resolution)
			}
			if pl.Audio != "" {
				buf.WriteString(",AUDIO=\"")
				buf.WriteString(pl.Audio)
				buf.WriteRune('"')
			}
			if pl.Video != "" {
				buf.WriteString(",VIDEO=\"")
				buf.WriteString(pl.Video)
				buf.WriteRune('"')
			}
			if pl.Captions != "" {
				buf.WriteString(",CLOSED-CAPTIONS=")
				if pl.Captions == "NONE" {
					buf.WriteString(pl.Captions) // CC should not be quoted when eq NONE
				} else {
					buf.WriteRune('"')
					buf.WriteString(pl.Captions)
					buf.WriteRune('"')
				}
			}
			if pl.Subtitles != "" {
				buf.WriteString(",SUBTITLES=\"")
				buf.WriteString(pl.Subtitles)
				buf.WriteRune('"')
			}
			if pl.Name != "" {
				buf.WriteString(",NAME=\"")
				buf.WriteString(pl.Name)
				buf.WriteRune('"')
			}
			if pl.FrameRate != 0 {
				buf.WriteString(",FRAME-RATE=")
				buf.WriteString(strconv.FormatFloat(pl.FrameRate, 'f', 3, 64))
			}
			if pl.VideoRange != "" {
				buf.WriteString(",VIDEO-RANGE=")
				buf.WriteString(pl.VideoRange)
			}
			if pl.HDCPLevel != "" {
				buf.WriteString(",HDCP-LEVEL=")
				buf.WriteString(pl.HDCPLevel)
			}

			buf.WriteRune('\n')
			buf.WriteString(pl.URI)
			if p.Args != "" {
				if strings.Contains(pl.URI, "?") {
					buf.WriteRune('&')
				} else {
					buf.WriteRune('?')
				}
				buf.WriteString(p.Args)
			}
			buf.WriteRune('\n')
		}
	}

	return
}

// SetCustomTag sets the provided tag on the master playlist for its TagName
func (p *MasterPlaylist) SetCustomTag(tag CustomTag) {
	if p.Custom == nil {
		p.Custom = make(map[string]CustomTag)
	}

	p.Custom[tag.TagName()] = tag
}

// Version returns the current playlist version number
func (p *MasterPlaylist) Version() uint8 {
	return p.ver
}

// SetVersion sets the playlist version number, note the version maybe changed
// automatically by other Set methods.
func (p *MasterPlaylist) SetVersion(ver uint8) {
	p.ver = ver
}

// IndependentSegments returns true if all media samples in a segment can be
// decoded without information from other segments.
func (p *MasterPlaylist) IndependentSegments() bool {
	return p.independentSegments
}

// SetIndependentSegments sets whether all media samples in a segment can be
// decoded without information from other segments.
func (p *MasterPlaylist) SetIndependentSegments(b bool) {
	p.independentSegments = b
}

// For compatibility with Stringer interface
// For example fmt.Printf("%s", sampleMediaList) will encode
// playist and print its string representation.
func (p *MasterPlaylist) String() string {
	return p.Encode().String()
}

// Creates new media playlist structure.
// Winsize defines how much items will displayed on playlist generation.
// Capacity is total size of a playlist.
func NewMediaPlaylist(winsize uint) *MediaPlaylist {
	p := &MediaPlaylist{
		ver:      minver,
		Segments: list.New(),
	}
	p.SetWinSize(winsize)
	return p
}

// Remove current segment from the head of chunk slice form a media playlist. Useful for sliding playlists.
// This operation does reset playlist cache.
func (p *MediaPlaylist) Remove() (err error) {
	if p.Segments.Len() == 0 {
		return errors.New("playlist is empty")
	}
	p.Segments.Remove(p.Segments.Front())
	if !p.Closed {
		p.SeqNo++
	}
	return nil
}

// Append general chunk to the tail of chunk slice for a media playlist.
// This operation does reset playlist cache.
func (p *MediaPlaylist) Append(uri string, duration float64, title string) error {
	seg := new(MediaSegment)
	seg.URI = uri
	seg.Duration = duration
	seg.Title = title
	return p.AppendSegment(seg)
}

// AppendSegment appends a MediaSegment to the tail of chunk slice for a media playlist.
// This operation does reset playlist cache.
func (p *MediaPlaylist) AppendSegment(seg *MediaSegment) error {
	seg.SeqId = p.SeqNo
	if p.Segments.Len() > 0 {
		seg.SeqId = p.Segments.Back().Value.(*MediaSegment).SeqId + 1
	}

	p.Segments.PushBack(seg)
	if p.TargetDuration < seg.Duration {
		p.TargetDuration = math.Ceil(seg.Duration)
	}
	return nil
}

// Combines two operations: firstly it removes one chunk from the head of chunk slice and move pointer to
// next chunk. Secondly it appends one chunk to the tail of chunk slice. Useful for sliding playlists.
// This operation does reset cache.
func (p *MediaPlaylist) Slide(uri string, duration float64, title string) {
	if !p.Closed && uint(p.Segments.Len()) >= p.winsize {
		p.Remove()
	}
	p.Append(uri, duration, title)
}

// Generate output in M3U8 format. Marshal `winsize` elements from bottom of the `segments` queue.
func (p *MediaPlaylist) Encode() (buf *bytes.Buffer) {
	buf = bytes.NewBuffer(nil)

	buf.WriteString("#EXTM3U\n#EXT-X-VERSION:")
	buf.WriteString(strver(p.ver))
	buf.WriteRune('\n')

	// Write any custom master tags
	if p.Custom != nil {
		for _, v := range p.Custom {
			if customBuf := v.Encode(); customBuf != nil {
				buf.WriteString(customBuf.String())
				buf.WriteRune('\n')
			}
		}
	}

	if p.MediaType > 0 {
		buf.WriteString("#EXT-X-PLAYLIST-TYPE:")
		switch p.MediaType {
		case EVENT:
			buf.WriteString("EVENT\n")
			buf.WriteString("#EXT-X-ALLOW-CACHE:NO\n")
		case VOD:
			buf.WriteString("VOD\n")
		}
	}
	buf.WriteString("#EXT-X-MEDIA-SEQUENCE:")
	buf.WriteString(strconv.FormatUint(p.SeqNo, 10))
	buf.WriteRune('\n')
	buf.WriteString("#EXT-X-TARGETDURATION:")
	buf.WriteString(strconv.FormatInt(int64(math.Ceil(p.TargetDuration)), 10)) // due section 3.4.2 of M3U8 specs EXT-X-TARGETDURATION must be integer
	buf.WriteRune('\n')
	if p.StartTime > 0.0 {
		buf.WriteString("#EXT-X-START:TIME-OFFSET=")
		buf.WriteString(strconv.FormatFloat(p.StartTime, 'f', -1, 64))
		if p.StartTimePrecise {
			buf.WriteString(",PRECISE=YES")
		}
		buf.WriteRune('\n')
	}
	if p.DiscontinuitySeq != 0 {
		buf.WriteString("#EXT-X-DISCONTINUITY-SEQUENCE:")
		buf.WriteString(strconv.FormatUint(uint64(p.DiscontinuitySeq), 10))
		buf.WriteRune('\n')
	}
	if p.Iframe {
		buf.WriteString("#EXT-X-I-FRAMES-ONLY\n")
	}
	// Widevine tags
	if p.WV != nil {
		if p.WV.AudioChannels != 0 {
			buf.WriteString("#WV-AUDIO-CHANNELS ")
			buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioChannels), 10))
			buf.WriteRune('\n')
		}
		if p.WV.AudioFormat != 0 {
			buf.WriteString("#WV-AUDIO-FORMAT ")
			buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioFormat), 10))
			buf.WriteRune('\n')
		}
		if p.WV.AudioProfileIDC != 0 {
			buf.WriteString("#WV-AUDIO-PROFILE-IDC ")
			buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioProfileIDC), 10))
			buf.WriteRune('\n')
		}
		if p.WV.AudioSampleSize != 0 {
			buf.WriteString("#WV-AUDIO-SAMPLE-SIZE ")
			buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioSampleSize), 10))
			buf.WriteRune('\n')
		}
		if p.WV.AudioSamplingFrequency != 0 {
			buf.WriteString("#WV-AUDIO-SAMPLING-FREQUENCY ")
			buf.WriteString(strconv.FormatUint(uint64(p.WV.AudioSamplingFrequency), 10))
			buf.WriteRune('\n')
		}
		if p.WV.CypherVersion != "" {
			buf.WriteString("#WV-CYPHER-VERSION ")
			buf.WriteString(p.WV.CypherVersion)
			buf.WriteRune('\n')
		}
		if p.WV.ECM != "" {
			buf.WriteString("#WV-ECM ")
			buf.WriteString(p.WV.ECM)
			buf.WriteRune('\n')
		}
		if p.WV.VideoFormat != 0 {
			buf.WriteString("#WV-VIDEO-FORMAT ")
			buf.WriteString(strconv.FormatUint(uint64(p.WV.VideoFormat), 10))
			buf.WriteRune('\n')
		}
		if p.WV.VideoFrameRate != 0 {
			buf.WriteString("#WV-VIDEO-FRAME-RATE ")
			buf.WriteString(strconv.FormatUint(uint64(p.WV.VideoFrameRate), 10))
			buf.WriteRune('\n')
		}
		if p.WV.VideoLevelIDC != 0 {
			buf.WriteString("#WV-VIDEO-LEVEL-IDC")
			buf.WriteString(strconv.FormatUint(uint64(p.WV.VideoLevelIDC), 10))
			buf.WriteRune('\n')
		}
		if p.WV.VideoProfileIDC != 0 {
			buf.WriteString("#WV-VIDEO-PROFILE-IDC ")
			buf.WriteString(strconv.FormatUint(uint64(p.WV.VideoProfileIDC), 10))
			buf.WriteRune('\n')
		}
		if p.WV.VideoResolution != "" {
			buf.WriteString("#WV-VIDEO-RESOLUTION ")
			buf.WriteString(p.WV.VideoResolution)
			buf.WriteRune('\n')
		}
		if p.WV.VideoSAR != "" {
			buf.WriteString("#WV-VIDEO-SAR ")
			buf.WriteString(p.WV.VideoSAR)
			buf.WriteRune('\n')
		}
	}

	var (
		seg           *MediaSegment
		lastMap       *Map
		lastKey       *Key
		durationCache = make(map[float64]string)
	)

	for e := p.Segments.Front(); e != nil; e = e.Next() {
		seg = e.Value.(*MediaSegment)
		if seg.SCTE != nil {
			switch seg.SCTE.Syntax {
			case SCTE35_67_2014:
				buf.WriteString("#EXT-SCTE35:")
				buf.WriteString("CUE=\"")
				buf.WriteString(seg.SCTE.Cue)
				buf.WriteRune('"')
				if seg.SCTE.ID != "" {
					buf.WriteString(",ID=\"")
					buf.WriteString(seg.SCTE.ID)
					buf.WriteRune('"')
				}
				if seg.SCTE.Time != 0 {
					buf.WriteString(",TIME=")
					buf.WriteString(strconv.FormatFloat(seg.SCTE.Time, 'f', -1, 64))
				}
				buf.WriteRune('\n')
			case SCTE35_OATCLS:
				switch seg.SCTE.CueType {
				case SCTE35Cue_Start:
					buf.WriteString("#EXT-OATCLS-SCTE35:")
					buf.WriteString(seg.SCTE.Cue)
					buf.WriteRune('\n')
					buf.WriteString("#EXT-X-CUE-OUT:")
					buf.WriteString(strconv.FormatFloat(seg.SCTE.Time, 'f', -1, 64))
					buf.WriteRune('\n')
				case SCTE35Cue_Mid:
					buf.WriteString("#EXT-X-CUE-OUT-CONT:")
					buf.WriteString("ElapsedTime=")
					buf.WriteString(strconv.FormatFloat(seg.SCTE.Elapsed, 'f', -1, 64))
					buf.WriteString(",Duration=")
					buf.WriteString(strconv.FormatFloat(seg.SCTE.Time, 'f', -1, 64))
					buf.WriteString(",SCTE35=")
					buf.WriteString(seg.SCTE.Cue)
					buf.WriteRune('\n')
				case SCTE35Cue_End:
					buf.WriteString("#EXT-X-CUE-IN")
					buf.WriteRune('\n')
				}
			}
		}
		for _, daterange := range seg.Dateranges {
			buf.WriteString("#EXT-X-DATERANGE:")
			buf.WriteString("ID=\"")
			buf.WriteString(daterange.ID)
			buf.WriteRune('"')
			if daterange.Class != nil {
				buf.WriteString(",CLASS=\"")
				buf.WriteString(*daterange.Class)
				buf.WriteRune('"')
			}
			buf.WriteString(",START-DATE=\"")
			buf.WriteString(daterange.StartDate.Format(DATETIME))
			buf.WriteRune('"')
			if daterange.EndDate != nil {
				buf.WriteString(",END-DATE=\"")
				buf.WriteString(daterange.EndDate.Format(DATETIME))
				buf.WriteRune('"')
			}
			if daterange.Duration != nil {
				buf.WriteString(",DURATION=")
				buf.WriteString(strconv.FormatFloat(daterange.Duration.Seconds(), 'f', -1, 64))
			}
			if daterange.PlannedDuration != nil {
				buf.WriteString(",PLANNED-DURATION=")
				buf.WriteString(strconv.FormatFloat(daterange.PlannedDuration.Seconds(), 'f', -1, 64))
			}
			for attr, value := range daterange.X {
				buf.WriteString(",X-")
				buf.WriteString(attr)
				buf.WriteString("=\"")
				buf.WriteString(value)
				buf.WriteRune('"')
			}
			if daterange.SCTE35Command != nil {
				buf.WriteString(",SCTE35-CMD=\"")
				buf.WriteString(*daterange.SCTE35Command)
				buf.WriteRune('"')
			}
			if daterange.SCTE35In != nil {
				buf.WriteString(",SCTE35-IN=\"")
				buf.WriteString(*daterange.SCTE35In)
				buf.WriteRune('"')
			}
			if daterange.SCTE35Out != nil {
				buf.WriteString(",SCTE35-OUT=\"")
				buf.WriteString(*daterange.SCTE35Out)
				buf.WriteRune('"')
			}
			if daterange.EndOnNext {
				buf.WriteString(",END-ON-NEXT=YES")
			}
			buf.WriteRune('\n')
		}

		if seg.Key != nil && (lastKey == nil || !seg.Key.Equal(lastKey)) {
			buf.WriteString("#EXT-X-KEY:")
			buf.WriteString("METHOD=")
			buf.WriteString(seg.Key.Method)
			if seg.Key.Method != "NONE" {
				buf.WriteString(",URI=\"")
				buf.WriteString(seg.Key.URI)
				buf.WriteRune('"')
				if seg.Key.IV != "" {
					buf.WriteString(",IV=")
					buf.WriteString(seg.Key.IV)
				}
				if seg.Key.Keyformat != "" {
					buf.WriteString(",KEYFORMAT=\"")
					buf.WriteString(seg.Key.Keyformat)
					buf.WriteRune('"')
				}
				if seg.Key.Keyformatversions != "" {
					buf.WriteString(",KEYFORMATVERSIONS=\"")
					buf.WriteString(seg.Key.Keyformatversions)
					buf.WriteRune('"')
				}
			}
			buf.WriteRune('\n')
		}
		lastKey = seg.Key

		if seg.Discontinuity {
			buf.WriteString("#EXT-X-DISCONTINUITY\n")
		}

		if seg.Map != nil && (lastMap == nil || !seg.Map.Equal(lastMap)) {
			buf.WriteString("#EXT-X-MAP:")
			buf.WriteString("URI=\"")
			buf.WriteString(seg.Map.URI)
			buf.WriteRune('"')
			if seg.Map.Limit > 0 {
				buf.WriteString(",BYTERANGE=")
				buf.WriteString(strconv.FormatInt(seg.Map.Limit, 10))
				buf.WriteRune('@')
				buf.WriteString(strconv.FormatInt(seg.Map.Offset, 10))
			}
			buf.WriteRune('\n')

		}
		lastMap = seg.Map

		if !seg.ProgramDateTime.IsZero() {
			buf.WriteString("#EXT-X-PROGRAM-DATE-TIME:")
			buf.WriteString(seg.ProgramDateTime.Format(DATETIME))
			buf.WriteRune('\n')
		}
		if seg.Limit > 0 {
			buf.WriteString("#EXT-X-BYTERANGE:")
			buf.WriteString(strconv.FormatInt(seg.Limit, 10))
			buf.WriteRune('@')
			buf.WriteString(strconv.FormatInt(seg.Offset, 10))
			buf.WriteRune('\n')
		}

		// Add Custom Segment Tags here
		if seg.Custom != nil {
			for _, v := range seg.Custom {
				if customBuf := v.Encode(); customBuf != nil {
					buf.WriteString(customBuf.String())
					buf.WriteRune('\n')
				}
			}
		}

		buf.WriteString("#EXTINF:")
		if str, ok := durationCache[seg.Duration]; ok {
			buf.WriteString(str)
		} else {
			if p.durationAsInt {
				// Old Android players has problems with non integer Duration.
				durationCache[seg.Duration] = strconv.FormatInt(int64(math.Ceil(seg.Duration)), 10)
			} else {
				// Wowza Mediaserver and some others prefer floats.
				durationCache[seg.Duration] = strconv.FormatFloat(seg.Duration, 'f', 3, 32)
			}
			buf.WriteString(durationCache[seg.Duration])
		}
		buf.WriteRune(',')
		buf.WriteString(seg.Title)
		buf.WriteRune('\n')
		buf.WriteString(seg.URI)
		if p.Args != "" {
			buf.WriteRune('?')
			buf.WriteString(p.Args)
		}
		buf.WriteRune('\n')
	}
	if p.Closed {
		buf.WriteString("#EXT-X-ENDLIST\n")
	}
	return
}

// For compatibility with Stringer interface
// For example fmt.Printf("%s", sampleMediaList) will encode
// playist and print its string representation.
func (p *MediaPlaylist) String() string {
	return p.Encode().String()
}

// TargetDuration will be int on Encode
func (p *MediaPlaylist) DurationAsInt(yes bool) {
	if yes {
		// duration must be integers if protocol version is less than 3
		version(&p.ver, 3)
	}
	p.durationAsInt = yes
}

// Count tells us the number of items that are currently in the media playlist
func (p *MediaPlaylist) Count() uint {
	return uint(p.Segments.Len())
}

// Close sliding playlist and make them fixed.
func (p *MediaPlaylist) Close() {
	p.Closed = true
}

// Mark medialist as consists of only I-frames (Intra frames).
// Set tag for the whole list.
func (p *MediaPlaylist) SetIframeOnly() {
	version(&p.ver, 4) // due section 4.3.3
	p.Iframe = true
}

// Set encryption key for the current segment of media playlist (pointer to Segment.Key)
func (p *MediaPlaylist) SetKey(method, uri, iv, keyformat, keyformatversions string) error {
	if p.Segments.Len() == 0 {
		return errors.New("playlist is empty")
	}

	// A Media Playlist MUST indicate a EXT-X-VERSION of 5 or higher if it
	// contains:
	//   - The KEYFORMAT and KEYFORMATVERSIONS attributes of the EXT-X-KEY tag.
	if keyformat != "" || keyformatversions != "" {
		version(&p.ver, 5)
	}

	p.last().Key = &Key{method, uri, iv, keyformat, keyformatversions}
	return nil
}

// Set map for the current segment of media playlist (pointer to Segment.Map)
func (p *MediaPlaylist) SetMap(uri string, limit, offset int64) error {
	if p.Segments.Len() == 0 {
		return errors.New("playlist is empty")
	}
	version(&p.ver, 5) // due section 4
	p.last().Map = &Map{uri, limit, offset}
	return nil
}

// Set limit and offset for the current media segment (EXT-X-BYTERANGE support for protocol version 4).
func (p *MediaPlaylist) SetRange(limit, offset int64) error {
	if p.Segments.Len() == 0 {
		return errors.New("playlist is empty")
	}
	version(&p.ver, 4) // due section 3.4.1
	p.last().Limit = limit
	p.last().Offset = offset
	return nil
}

// SetSCTE sets the SCTE cue format for the current media segment.
//
// Deprecated: Use SetSCTE35 instead.
func (p *MediaPlaylist) SetSCTE(cue string, id string, time float64) error {
	return p.SetSCTE35(&SCTE{Syntax: SCTE35_67_2014, Cue: cue, ID: id, Time: time})
}

// SetSCTE35 sets the SCTE cue format for the current media segment
func (p *MediaPlaylist) SetSCTE35(scte35 *SCTE) error {
	if p.Segments.Len() == 0 {
		return errors.New("playlist is empty")
	}
	p.last().SCTE = scte35
	return nil
}

// SetDaterange sets the Daterange the current media segment
func (p *MediaPlaylist) SetDateranges(dateranges []*Daterange) error {
	if p.Segments.Len() == 0 {
		return errors.New("playlist is empty")
	}
	p.last().Dateranges = dateranges
	return nil
}

// Set discontinuity flag for the current media segment.
// EXT-X-DISCONTINUITY indicates an encoding discontinuity between the media segment
// that follows it and the one that preceded it (i.e. file format, number and type of tracks,
// encoding parameters, encoding sequence, timestamp sequence).
func (p *MediaPlaylist) SetDiscontinuity() error {
	if p.Segments.Len() == 0 {
		return errors.New("playlist is empty")
	}
	p.last().Discontinuity = true
	return nil
}

// Set program date and time for the current media segment.
// EXT-X-PROGRAM-DATE-TIME tag associates the first sample of a
// media segment with an absolute date and/or time.  It applies only
// to the current media segment.
// Date/time format is YYYY-MM-DDThh:mm:ssZ (ISO8601) and includes time zone.
func (p *MediaPlaylist) SetProgramDateTime(value time.Time) error {
	if p.Segments.Len() == 0 {
		return errors.New("playlist is empty")
	}
	p.last().ProgramDateTime = value
	return nil
}

// SetCustomTag sets the provided tag on the media playlist for its TagName
func (p *MediaPlaylist) SetCustomTag(tag CustomTag) {
	if p.Custom == nil {
		p.Custom = make(map[string]CustomTag)
	}

	p.Custom[tag.TagName()] = tag
}

// SetCustomTag sets the provided tag on the current media segment for its TagName
func (p *MediaPlaylist) SetCustomSegmentTag(tag CustomTag) error {
	if p.Segments.Len() == 0 {
		return errors.New("playlist is empty")
	}

	last := p.last()

	if last.Custom == nil {
		last.Custom = make(map[string]CustomTag)
	}

	last.Custom[tag.TagName()] = tag

	return nil
}

// Version returns the current playlist version number
func (p *MediaPlaylist) Version() uint8 {
	return p.ver
}

// SetVersion sets the playlist version number, note the version maybe changed
// automatically by other Set methods.
func (p *MediaPlaylist) SetVersion(ver uint8) {
	p.ver = ver
}

// WinSize returns the playlist's window size.
func (p *MediaPlaylist) WinSize() uint {
	return p.winsize
}

// SetWinSize overwrites the playlist's window size.
func (p *MediaPlaylist) SetWinSize(winsize uint) {
	p.winsize = winsize
}
