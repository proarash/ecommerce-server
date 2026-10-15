package export

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const (
	XLSXMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	MaxRows  = 50000

	minColWidth   = 14.0
	maxColWidth   = 60.0
	widthFactor   = 1.3
	widthPadding  = 4.0
	moneyNumFmt   = 3
	headerFill    = "DCE6FF"
	footerFill    = "F2F2F2"
	borderColor   = "999999"
	timeLayout    = "2006-01-02 15:04"
	headerRowSize = 24.0
)

var site struct {
	name string
	url  string
}

func SetSite(name, url string) {
	site.name, site.url = name, url
}

type Sheet struct {
	Title     string
	Headers   []string
	Rows      [][]any
	Footer    [][]any
	Notes     [][]any
	MoneyCols []int
}

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(timeLayout)
}

var jalaliMonthOffsets = [12]int{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}

func toJalali(gy, gm, gd int) (int, int, int) {
	gy2 := gy
	if gm > 2 {
		gy2 = gy + 1
	}
	days := 355666 + 365*gy + (gy2+3)/4 - (gy2+99)/100 + (gy2+399)/400 + gd + jalaliMonthOffsets[gm-1]
	jy := -1595 + 33*(days/12053)
	days %= 12053
	jy += 4 * (days / 1461)
	days %= 1461
	if days > 365 {
		jy += (days - 1) / 365
		days = (days - 1) % 365
	}
	if days < 186 {
		return jy, 1 + days/31, 1 + days%31
	}
	return jy, 7 + (days-186)/30, 1 + (days-186)%30
}

func FormatJalali(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	t = t.Local()
	y, m, d := toJalali(t.Year(), int(t.Month()), t.Day())
	return fmt.Sprintf("%04d/%02d/%02d %02d:%02d", y, m, d, t.Hour(), t.Minute())
}

func FormatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return FormatTime(*t)
}

func sheetName(title string, index int) string {
	name := strings.NewReplacer(":", " ", "\\", " ", "/", " ", "?", " ", "*", " ", "[", " ", "]", " ").Replace(title)
	if name = strings.TrimSpace(name); name == "" {
		name = fmt.Sprintf("Sheet%d", index+1)
	}
	if utf8.RuneCountInString(name) > 31 {
		name = string([]rune(name)[:31])
	}
	return name
}

func textWidth(value any) float64 {
	return float64(utf8.RuneCountInString(fmt.Sprint(value)))*widthFactor + widthPadding
}

func isEmpty(value any) bool {
	text, ok := value.(string)
	return ok && text == ""
}

type styles struct {
	header, text, money, noteLabel, noteValue, footerLabel, footerMoney int
}

func buildStyles(f *excelize.File) (styles, error) {
	border := []excelize.Border{
		{Type: "left", Color: borderColor, Style: 1},
		{Type: "right", Color: borderColor, Style: 1},
		{Type: "top", Color: borderColor, Style: 1},
		{Type: "bottom", Color: borderColor, Style: 1},
	}
	center := &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}
	right := &excelize.Alignment{Horizontal: "right", Vertical: "center"}
	bold := &excelize.Font{Bold: true}
	var s styles
	var err error
	for _, def := range []struct {
		target *int
		style  excelize.Style
	}{
		{&s.header, excelize.Style{Font: bold, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{headerFill}}, Alignment: center, Border: border}},
		{&s.text, excelize.Style{Alignment: center, Border: border}},
		{&s.money, excelize.Style{Alignment: center, Border: border, NumFmt: moneyNumFmt}},
		{&s.noteLabel, excelize.Style{Font: bold, Alignment: right}},
		{&s.noteValue, excelize.Style{Alignment: right, NumFmt: moneyNumFmt}},
		{&s.footerLabel, excelize.Style{Font: bold, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{footerFill}}, Alignment: center, Border: border}},
		{&s.footerMoney, excelize.Style{Font: bold, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{footerFill}}, Alignment: center, Border: border, NumFmt: moneyNumFmt}},
	} {
		style := def.style
		if *def.target, err = f.NewStyle(&style); err != nil {
			return s, err
		}
	}
	return s, nil
}

func Build(sheets ...Sheet) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	rtl := true
	st, err := buildStyles(f)
	if err != nil {
		return nil, err
	}

	for i, sheet := range sheets {
		name := sheetName(sheet.Title, i)
		if i == 0 {
			if err := f.SetSheetName(f.GetSheetName(0), name); err != nil {
				return nil, err
			}
		} else if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
		if err := f.SetSheetView(name, 0, &excelize.ViewOptions{RightToLeft: &rtl}); err != nil {
			return nil, err
		}

		cols := len(sheet.Headers)
		widths := make([]float64, cols)
		grow := func(values []any) {
			for c, value := range values {
				if c < cols && textWidth(value) > widths[c] {
					widths[c] = textWidth(value)
				}
			}
		}
		for c, header := range sheet.Headers {
			widths[c] = textWidth(header)
		}

		row := 1
		for _, note := range sheet.Notes {
			cell, _ := excelize.CoordinatesToCellName(1, row)
			if err := f.SetSheetRow(name, cell, &note); err != nil {
				return nil, err
			}
			if len(note) > 0 {
				if err := f.SetCellStyle(name, cell, cell, st.noteLabel); err != nil {
					return nil, err
				}
			}
			if len(note) > 1 {
				value, _ := excelize.CoordinatesToCellName(2, row)
				if err := f.SetCellStyle(name, value, value, st.noteValue); err != nil {
					return nil, err
				}
			}
			row++
		}
		if len(sheet.Notes) > 0 {
			row++
		}

		headerRow := row
		start, _ := excelize.CoordinatesToCellName(1, headerRow)
		end, _ := excelize.CoordinatesToCellName(cols, headerRow)
		if err := f.SetSheetRow(name, start, &sheet.Headers); err != nil {
			return nil, err
		}
		if err := f.SetCellStyle(name, start, end, st.header); err != nil {
			return nil, err
		}
		if err := f.SetRowHeight(name, headerRow, headerRowSize); err != nil {
			return nil, err
		}
		row++

		for _, values := range sheet.Rows {
			first, _ := excelize.CoordinatesToCellName(1, row)
			last, _ := excelize.CoordinatesToCellName(cols, row)
			if err := f.SetSheetRow(name, first, &values); err != nil {
				return nil, err
			}
			if err := f.SetCellStyle(name, first, last, st.text); err != nil {
				return nil, err
			}
			for _, c := range sheet.MoneyCols {
				cell, _ := excelize.CoordinatesToCellName(c+1, row)
				if err := f.SetCellStyle(name, cell, cell, st.money); err != nil {
					return nil, err
				}
			}
			grow(values)
			row++
		}

		for _, values := range sheet.Footer {
			first, _ := excelize.CoordinatesToCellName(1, row)
			if err := f.SetSheetRow(name, first, &values); err != nil {
				return nil, err
			}
			for c, value := range values {
				if isEmpty(value) {
					continue
				}
				cell, _ := excelize.CoordinatesToCellName(c+1, row)
				style := st.footerLabel
				if slices.Contains(sheet.MoneyCols, c) {
					style = st.footerMoney
				}
				if err := f.SetCellStyle(name, cell, cell, style); err != nil {
					return nil, err
				}
			}
			grow(values)
			row++
		}

		metaRow := row + 1
		meta := [][]any{
			{"وب‌سایت", site.name},
			{"آدرس", site.url},
			{"تاریخ تهیه گزارش", FormatTime(time.Now())},
		}
		for _, values := range meta {
			first, _ := excelize.CoordinatesToCellName(1, metaRow)
			if err := f.SetSheetRow(name, first, &values); err != nil {
				return nil, err
			}
			if err := f.SetCellStyle(name, first, first, st.noteLabel); err != nil {
				return nil, err
			}
			value, _ := excelize.CoordinatesToCellName(2, metaRow)
			if err := f.SetCellStyle(name, value, value, st.noteValue); err != nil {
				return nil, err
			}
			metaRow++
		}
		if site.url != "" {
			link, _ := excelize.CoordinatesToCellName(2, row+2)
			if err := f.SetCellHyperLink(name, link, site.url, "External"); err != nil {
				return nil, err
			}
		}

		for c, width := range widths {
			col, _ := excelize.ColumnNumberToName(c + 1)
			width = min(max(width, minColWidth), maxColWidth)
			if err := f.SetColWidth(name, col, col, width); err != nil {
				return nil, err
			}
		}
		if err := f.SetPanes(name, &excelize.Panes{
			Freeze:      true,
			YSplit:      headerRow,
			TopLeftCell: fmt.Sprintf("A%d", headerRow+1),
			ActivePane:  "bottomLeft",
		}); err != nil {
			return nil, err
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Send(c *gin.Context, filename string, sheets ...Sheet) {
	data, err := Build(sheets...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not build the excel file"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, XLSXMime, data)
}

func Filename(prefix string) string {
	return fmt.Sprintf("%s-%s.xlsx", prefix, time.Now().Format("2006-01-02_15-04-05"))
}
