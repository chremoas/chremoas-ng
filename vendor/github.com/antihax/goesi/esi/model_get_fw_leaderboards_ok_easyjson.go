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

func easyjson38812470DecodeGithubComAntihaxGoesiEsi(in *jlexer.Lexer, out *GetFwLeaderboardsOkList) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		in.Skip()
		*out = nil
	} else {
		in.Delim('[')
		if *out == nil {
			if !in.IsDelim(']') {
				*out = make(GetFwLeaderboardsOkList, 0, 0)
			} else {
				*out = GetFwLeaderboardsOkList{}
			}
		} else {
			*out = (*out)[:0]
		}
		for !in.IsDelim(']') {
			var v1 GetFwLeaderboardsOk
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
func easyjson38812470EncodeGithubComAntihaxGoesiEsi(out *jwriter.Writer, in GetFwLeaderboardsOkList) {
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
func (v GetFwLeaderboardsOkList) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson38812470EncodeGithubComAntihaxGoesiEsi(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v GetFwLeaderboardsOkList) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson38812470EncodeGithubComAntihaxGoesiEsi(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *GetFwLeaderboardsOkList) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson38812470DecodeGithubComAntihaxGoesiEsi(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *GetFwLeaderboardsOkList) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson38812470DecodeGithubComAntihaxGoesiEsi(l, v)
}
func easyjson38812470DecodeGithubComAntihaxGoesiEsi1(in *jlexer.Lexer, out *GetFwLeaderboardsOk) {
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
		case "kills":
			easyjson38812470DecodeGithubComAntihaxGoesiEsi2(in, &out.Kills)
		case "victory_points":
			easyjson38812470DecodeGithubComAntihaxGoesiEsi3(in, &out.VictoryPoints)
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
func easyjson38812470EncodeGithubComAntihaxGoesiEsi1(out *jwriter.Writer, in GetFwLeaderboardsOk) {
	out.RawByte('{')
	first := true
	_ = first
	if true {
		const prefix string = ",\"kills\":"
		first = false
		out.RawString(prefix[1:])
		easyjson38812470EncodeGithubComAntihaxGoesiEsi2(out, in.Kills)
	}
	if true {
		const prefix string = ",\"victory_points\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		easyjson38812470EncodeGithubComAntihaxGoesiEsi3(out, in.VictoryPoints)
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v GetFwLeaderboardsOk) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson38812470EncodeGithubComAntihaxGoesiEsi1(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v GetFwLeaderboardsOk) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson38812470EncodeGithubComAntihaxGoesiEsi1(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *GetFwLeaderboardsOk) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson38812470DecodeGithubComAntihaxGoesiEsi1(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *GetFwLeaderboardsOk) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson38812470DecodeGithubComAntihaxGoesiEsi1(l, v)
}
func easyjson38812470DecodeGithubComAntihaxGoesiEsi3(in *jlexer.Lexer, out *GetFwLeaderboardsVictoryPoints) {
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
		case "active_total":
			if in.IsNull() {
				in.Skip()
				out.ActiveTotal = nil
			} else {
				in.Delim('[')
				if out.ActiveTotal == nil {
					if !in.IsDelim(']') {
						out.ActiveTotal = make([]GetFwLeaderboardsActiveTotalActiveTotal1, 0, 8)
					} else {
						out.ActiveTotal = []GetFwLeaderboardsActiveTotalActiveTotal1{}
					}
				} else {
					out.ActiveTotal = (out.ActiveTotal)[:0]
				}
				for !in.IsDelim(']') {
					var v4 GetFwLeaderboardsActiveTotalActiveTotal1
					(v4).UnmarshalEasyJSON(in)
					out.ActiveTotal = append(out.ActiveTotal, v4)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "last_week":
			if in.IsNull() {
				in.Skip()
				out.LastWeek = nil
			} else {
				in.Delim('[')
				if out.LastWeek == nil {
					if !in.IsDelim(']') {
						out.LastWeek = make([]GetFwLeaderboardsLastWeekLastWeek1, 0, 8)
					} else {
						out.LastWeek = []GetFwLeaderboardsLastWeekLastWeek1{}
					}
				} else {
					out.LastWeek = (out.LastWeek)[:0]
				}
				for !in.IsDelim(']') {
					var v5 GetFwLeaderboardsLastWeekLastWeek1
					easyjson38812470DecodeGithubComAntihaxGoesiEsi4(in, &v5)
					out.LastWeek = append(out.LastWeek, v5)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "yesterday":
			if in.IsNull() {
				in.Skip()
				out.Yesterday = nil
			} else {
				in.Delim('[')
				if out.Yesterday == nil {
					if !in.IsDelim(']') {
						out.Yesterday = make([]GetFwLeaderboardsYesterdayYesterday1, 0, 8)
					} else {
						out.Yesterday = []GetFwLeaderboardsYesterdayYesterday1{}
					}
				} else {
					out.Yesterday = (out.Yesterday)[:0]
				}
				for !in.IsDelim(']') {
					var v6 GetFwLeaderboardsYesterdayYesterday1
					easyjson38812470DecodeGithubComAntihaxGoesiEsi5(in, &v6)
					out.Yesterday = append(out.Yesterday, v6)
					in.WantComma()
				}
				in.Delim(']')
			}
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
func easyjson38812470EncodeGithubComAntihaxGoesiEsi3(out *jwriter.Writer, in GetFwLeaderboardsVictoryPoints) {
	out.RawByte('{')
	first := true
	_ = first
	if len(in.ActiveTotal) != 0 {
		const prefix string = ",\"active_total\":"
		first = false
		out.RawString(prefix[1:])
		{
			out.RawByte('[')
			for v7, v8 := range in.ActiveTotal {
				if v7 > 0 {
					out.RawByte(',')
				}
				(v8).MarshalEasyJSON(out)
			}
			out.RawByte(']')
		}
	}
	if len(in.LastWeek) != 0 {
		const prefix string = ",\"last_week\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v9, v10 := range in.LastWeek {
				if v9 > 0 {
					out.RawByte(',')
				}
				easyjson38812470EncodeGithubComAntihaxGoesiEsi4(out, v10)
			}
			out.RawByte(']')
		}
	}
	if len(in.Yesterday) != 0 {
		const prefix string = ",\"yesterday\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v11, v12 := range in.Yesterday {
				if v11 > 0 {
					out.RawByte(',')
				}
				easyjson38812470EncodeGithubComAntihaxGoesiEsi5(out, v12)
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}
func easyjson38812470DecodeGithubComAntihaxGoesiEsi5(in *jlexer.Lexer, out *GetFwLeaderboardsYesterdayYesterday1) {
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
		case "amount":
			out.Amount = int32(in.Int32())
		case "faction_id":
			out.FactionId = int32(in.Int32())
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
func easyjson38812470EncodeGithubComAntihaxGoesiEsi5(out *jwriter.Writer, in GetFwLeaderboardsYesterdayYesterday1) {
	out.RawByte('{')
	first := true
	_ = first
	if in.Amount != 0 {
		const prefix string = ",\"amount\":"
		first = false
		out.RawString(prefix[1:])
		out.Int32(int32(in.Amount))
	}
	if in.FactionId != 0 {
		const prefix string = ",\"faction_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int32(int32(in.FactionId))
	}
	out.RawByte('}')
}
func easyjson38812470DecodeGithubComAntihaxGoesiEsi4(in *jlexer.Lexer, out *GetFwLeaderboardsLastWeekLastWeek1) {
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
		case "amount":
			out.Amount = int32(in.Int32())
		case "faction_id":
			out.FactionId = int32(in.Int32())
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
func easyjson38812470EncodeGithubComAntihaxGoesiEsi4(out *jwriter.Writer, in GetFwLeaderboardsLastWeekLastWeek1) {
	out.RawByte('{')
	first := true
	_ = first
	if in.Amount != 0 {
		const prefix string = ",\"amount\":"
		first = false
		out.RawString(prefix[1:])
		out.Int32(int32(in.Amount))
	}
	if in.FactionId != 0 {
		const prefix string = ",\"faction_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int32(int32(in.FactionId))
	}
	out.RawByte('}')
}
func easyjson38812470DecodeGithubComAntihaxGoesiEsi2(in *jlexer.Lexer, out *GetFwLeaderboardsKills) {
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
		case "active_total":
			if in.IsNull() {
				in.Skip()
				out.ActiveTotal = nil
			} else {
				in.Delim('[')
				if out.ActiveTotal == nil {
					if !in.IsDelim(']') {
						out.ActiveTotal = make([]GetFwLeaderboardsActiveTotalActiveTotal, 0, 8)
					} else {
						out.ActiveTotal = []GetFwLeaderboardsActiveTotalActiveTotal{}
					}
				} else {
					out.ActiveTotal = (out.ActiveTotal)[:0]
				}
				for !in.IsDelim(']') {
					var v13 GetFwLeaderboardsActiveTotalActiveTotal
					(v13).UnmarshalEasyJSON(in)
					out.ActiveTotal = append(out.ActiveTotal, v13)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "last_week":
			if in.IsNull() {
				in.Skip()
				out.LastWeek = nil
			} else {
				in.Delim('[')
				if out.LastWeek == nil {
					if !in.IsDelim(']') {
						out.LastWeek = make([]GetFwLeaderboardsLastWeekLastWeek, 0, 8)
					} else {
						out.LastWeek = []GetFwLeaderboardsLastWeekLastWeek{}
					}
				} else {
					out.LastWeek = (out.LastWeek)[:0]
				}
				for !in.IsDelim(']') {
					var v14 GetFwLeaderboardsLastWeekLastWeek
					easyjson38812470DecodeGithubComAntihaxGoesiEsi6(in, &v14)
					out.LastWeek = append(out.LastWeek, v14)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "yesterday":
			if in.IsNull() {
				in.Skip()
				out.Yesterday = nil
			} else {
				in.Delim('[')
				if out.Yesterday == nil {
					if !in.IsDelim(']') {
						out.Yesterday = make([]GetFwLeaderboardsYesterdayYesterday, 0, 8)
					} else {
						out.Yesterday = []GetFwLeaderboardsYesterdayYesterday{}
					}
				} else {
					out.Yesterday = (out.Yesterday)[:0]
				}
				for !in.IsDelim(']') {
					var v15 GetFwLeaderboardsYesterdayYesterday
					easyjson38812470DecodeGithubComAntihaxGoesiEsi7(in, &v15)
					out.Yesterday = append(out.Yesterday, v15)
					in.WantComma()
				}
				in.Delim(']')
			}
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
func easyjson38812470EncodeGithubComAntihaxGoesiEsi2(out *jwriter.Writer, in GetFwLeaderboardsKills) {
	out.RawByte('{')
	first := true
	_ = first
	if len(in.ActiveTotal) != 0 {
		const prefix string = ",\"active_total\":"
		first = false
		out.RawString(prefix[1:])
		{
			out.RawByte('[')
			for v16, v17 := range in.ActiveTotal {
				if v16 > 0 {
					out.RawByte(',')
				}
				(v17).MarshalEasyJSON(out)
			}
			out.RawByte(']')
		}
	}
	if len(in.LastWeek) != 0 {
		const prefix string = ",\"last_week\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v18, v19 := range in.LastWeek {
				if v18 > 0 {
					out.RawByte(',')
				}
				easyjson38812470EncodeGithubComAntihaxGoesiEsi6(out, v19)
			}
			out.RawByte(']')
		}
	}
	if len(in.Yesterday) != 0 {
		const prefix string = ",\"yesterday\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v20, v21 := range in.Yesterday {
				if v20 > 0 {
					out.RawByte(',')
				}
				easyjson38812470EncodeGithubComAntihaxGoesiEsi7(out, v21)
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}
func easyjson38812470DecodeGithubComAntihaxGoesiEsi7(in *jlexer.Lexer, out *GetFwLeaderboardsYesterdayYesterday) {
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
		case "amount":
			out.Amount = int32(in.Int32())
		case "faction_id":
			out.FactionId = int32(in.Int32())
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
func easyjson38812470EncodeGithubComAntihaxGoesiEsi7(out *jwriter.Writer, in GetFwLeaderboardsYesterdayYesterday) {
	out.RawByte('{')
	first := true
	_ = first
	if in.Amount != 0 {
		const prefix string = ",\"amount\":"
		first = false
		out.RawString(prefix[1:])
		out.Int32(int32(in.Amount))
	}
	if in.FactionId != 0 {
		const prefix string = ",\"faction_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int32(int32(in.FactionId))
	}
	out.RawByte('}')
}
func easyjson38812470DecodeGithubComAntihaxGoesiEsi6(in *jlexer.Lexer, out *GetFwLeaderboardsLastWeekLastWeek) {
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
		case "amount":
			out.Amount = int32(in.Int32())
		case "faction_id":
			out.FactionId = int32(in.Int32())
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
func easyjson38812470EncodeGithubComAntihaxGoesiEsi6(out *jwriter.Writer, in GetFwLeaderboardsLastWeekLastWeek) {
	out.RawByte('{')
	first := true
	_ = first
	if in.Amount != 0 {
		const prefix string = ",\"amount\":"
		first = false
		out.RawString(prefix[1:])
		out.Int32(int32(in.Amount))
	}
	if in.FactionId != 0 {
		const prefix string = ",\"faction_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int32(int32(in.FactionId))
	}
	out.RawByte('}')
}
