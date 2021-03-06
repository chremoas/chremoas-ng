// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package esi

import (
	json "encoding/json"

	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonAa81674fDecodeGithubComAntihaxGoesiEsi(in *jlexer.Lexer, out *GetCorporationsCorporationIdStarbasesStarbaseIdOkList) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		in.Skip()
		*out = nil
	} else {
		in.Delim('[')
		if *out == nil {
			if !in.IsDelim(']') {
				*out = make(GetCorporationsCorporationIdStarbasesStarbaseIdOkList, 0, 0)
			} else {
				*out = GetCorporationsCorporationIdStarbasesStarbaseIdOkList{}
			}
		} else {
			*out = (*out)[:0]
		}
		for !in.IsDelim(']') {
			var v1 GetCorporationsCorporationIdStarbasesStarbaseIdOk
			(v1).UnmarshalEasyJSON(in)
			*out = append(*out, v1)
			in.WantComma()
		}
		in.Delim(']')
	}
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonAa81674fEncodeGithubComAntihaxGoesiEsi(out *jwriter.Writer, in GetCorporationsCorporationIdStarbasesStarbaseIdOkList) {
	if in == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
		out.RawString("null")
	} else {
		out.RawByte('[')
		for v2, v3 := range in {
			if v2 > 0 {
				out.RawByte(',')
			}
			(v3).MarshalEasyJSON(out)
		}
		out.RawByte(']')
	}
}

// MarshalJSON supports json.Marshaler interface
func (v GetCorporationsCorporationIdStarbasesStarbaseIdOkList) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonAa81674fEncodeGithubComAntihaxGoesiEsi(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v GetCorporationsCorporationIdStarbasesStarbaseIdOkList) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonAa81674fEncodeGithubComAntihaxGoesiEsi(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *GetCorporationsCorporationIdStarbasesStarbaseIdOkList) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonAa81674fDecodeGithubComAntihaxGoesiEsi(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *GetCorporationsCorporationIdStarbasesStarbaseIdOkList) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonAa81674fDecodeGithubComAntihaxGoesiEsi(l, v)
}
func easyjsonAa81674fDecodeGithubComAntihaxGoesiEsi1(in *jlexer.Lexer, out *GetCorporationsCorporationIdStarbasesStarbaseIdOk) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "allow_alliance_members":
			out.AllowAllianceMembers = bool(in.Bool())
		case "allow_corporation_members":
			out.AllowCorporationMembers = bool(in.Bool())
		case "anchor":
			out.Anchor = string(in.String())
		case "attack_if_at_war":
			out.AttackIfAtWar = bool(in.Bool())
		case "attack_if_other_security_status_dropping":
			out.AttackIfOtherSecurityStatusDropping = bool(in.Bool())
		case "attack_security_status_threshold":
			out.AttackSecurityStatusThreshold = float32(in.Float32())
		case "attack_standing_threshold":
			out.AttackStandingThreshold = float32(in.Float32())
		case "fuel_bay_take":
			out.FuelBayTake = string(in.String())
		case "fuel_bay_view":
			out.FuelBayView = string(in.String())
		case "fuels":
			if in.IsNull() {
				in.Skip()
				out.Fuels = nil
			} else {
				in.Delim('[')
				if out.Fuels == nil {
					if !in.IsDelim(']') {
						out.Fuels = make([]GetCorporationsCorporationIdStarbasesStarbaseIdFuel, 0, 8)
					} else {
						out.Fuels = []GetCorporationsCorporationIdStarbasesStarbaseIdFuel{}
					}
				} else {
					out.Fuels = (out.Fuels)[:0]
				}
				for !in.IsDelim(']') {
					var v4 GetCorporationsCorporationIdStarbasesStarbaseIdFuel
					easyjsonAa81674fDecodeGithubComAntihaxGoesiEsi2(in, &v4)
					out.Fuels = append(out.Fuels, v4)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "offline":
			out.Offline = string(in.String())
		case "online":
			out.Online = string(in.String())
		case "unanchor":
			out.Unanchor = string(in.String())
		case "use_alliance_standings":
			out.UseAllianceStandings = bool(in.Bool())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonAa81674fEncodeGithubComAntihaxGoesiEsi1(out *jwriter.Writer, in GetCorporationsCorporationIdStarbasesStarbaseIdOk) {
	out.RawByte('{')
	first := true
	_ = first
	if in.AllowAllianceMembers {
		const prefix string = ",\"allow_alliance_members\":"
		first = false
		out.RawString(prefix[1:])
		out.Bool(bool(in.AllowAllianceMembers))
	}
	if in.AllowCorporationMembers {
		const prefix string = ",\"allow_corporation_members\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.AllowCorporationMembers))
	}
	if in.Anchor != "" {
		const prefix string = ",\"anchor\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Anchor))
	}
	if in.AttackIfAtWar {
		const prefix string = ",\"attack_if_at_war\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.AttackIfAtWar))
	}
	if in.AttackIfOtherSecurityStatusDropping {
		const prefix string = ",\"attack_if_other_security_status_dropping\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.AttackIfOtherSecurityStatusDropping))
	}
	if in.AttackSecurityStatusThreshold != 0 {
		const prefix string = ",\"attack_security_status_threshold\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Float32(float32(in.AttackSecurityStatusThreshold))
	}
	if in.AttackStandingThreshold != 0 {
		const prefix string = ",\"attack_standing_threshold\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Float32(float32(in.AttackStandingThreshold))
	}
	if in.FuelBayTake != "" {
		const prefix string = ",\"fuel_bay_take\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.FuelBayTake))
	}
	if in.FuelBayView != "" {
		const prefix string = ",\"fuel_bay_view\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.FuelBayView))
	}
	if len(in.Fuels) != 0 {
		const prefix string = ",\"fuels\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v5, v6 := range in.Fuels {
				if v5 > 0 {
					out.RawByte(',')
				}
				easyjsonAa81674fEncodeGithubComAntihaxGoesiEsi2(out, v6)
			}
			out.RawByte(']')
		}
	}
	if in.Offline != "" {
		const prefix string = ",\"offline\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Offline))
	}
	if in.Online != "" {
		const prefix string = ",\"online\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Online))
	}
	if in.Unanchor != "" {
		const prefix string = ",\"unanchor\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Unanchor))
	}
	if in.UseAllianceStandings {
		const prefix string = ",\"use_alliance_standings\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.UseAllianceStandings))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v GetCorporationsCorporationIdStarbasesStarbaseIdOk) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonAa81674fEncodeGithubComAntihaxGoesiEsi1(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v GetCorporationsCorporationIdStarbasesStarbaseIdOk) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonAa81674fEncodeGithubComAntihaxGoesiEsi1(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *GetCorporationsCorporationIdStarbasesStarbaseIdOk) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonAa81674fDecodeGithubComAntihaxGoesiEsi1(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *GetCorporationsCorporationIdStarbasesStarbaseIdOk) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonAa81674fDecodeGithubComAntihaxGoesiEsi1(l, v)
}
func easyjsonAa81674fDecodeGithubComAntihaxGoesiEsi2(in *jlexer.Lexer, out *GetCorporationsCorporationIdStarbasesStarbaseIdFuel) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "quantity":
			out.Quantity = int32(in.Int32())
		case "type_id":
			out.TypeId = int32(in.Int32())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonAa81674fEncodeGithubComAntihaxGoesiEsi2(out *jwriter.Writer, in GetCorporationsCorporationIdStarbasesStarbaseIdFuel) {
	out.RawByte('{')
	first := true
	_ = first
	if in.Quantity != 0 {
		const prefix string = ",\"quantity\":"
		first = false
		out.RawString(prefix[1:])
		out.Int32(int32(in.Quantity))
	}
	if in.TypeId != 0 {
		const prefix string = ",\"type_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int32(int32(in.TypeId))
	}
	out.RawByte('}')
}
