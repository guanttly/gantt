import * as XLSX from 'xlsx-js-style'

interface DayShift {
  staff: string[]
  staffIds: string[]
  requiredCount: number
  actualCount: number
}

interface ShiftDraft {
  shiftId: string
  priority?: number
  days: Record<string, DayShift>
}

interface ShiftInfo {
  id: string
  name: string
  startTime?: string
  endTime?: string
  isOvernight?: boolean
  type?: string
}

interface StaffInfo {
  id: string
  name: string
  employeeId?: string
}

export interface MultiShiftScheduleData {
  startDate: string
  endDate: string
  shifts: Record<string, ShiftDraft>
  shiftInfoList: ShiftInfo[]
  staffList?: StaffInfo[]
}

interface DayEntry {
  shiftName: string
  startTime: string
  isOvertime: boolean
}

function parseDateLocal(dateStr: string): Date {
  const [year, month, day] = dateStr.split('-').map(Number)
  return new Date(year, month - 1, day)
}

function formatDateHeader(dateStr: string): string {
  const date = parseDateLocal(dateStr)
  const [, month, day] = dateStr.split('-').map(Number)
  const WEEKDAY = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return `${String(month).padStart(2, '0')}.${String(day).padStart(2, '0')}${WEEKDAY[date.getDay()]}`
}

function isNightShift(startTime: string | undefined): boolean {
  if (!startTime)
    return false
  const [h] = startTime.split(':').map(Number)
  return h >= 18
}

function isWeekend(dateStr: string): boolean {
  const wd = parseDateLocal(dateStr).getDay()
  return wd === 0 || wd === 6
}

function getDatesInRange(startDate: string, endDate: string): string[] {
  const dates: string[] = []
  const start = parseDateLocal(startDate)
  const end = parseDateLocal(endDate)
  const cur = new Date(start)
  while (cur.getTime() <= end.getTime()) {
    const y = cur.getFullYear()
    const m = String(cur.getMonth() + 1).padStart(2, '0')
    const d = String(cur.getDate()).padStart(2, '0')
    dates.push(`${y}-${m}-${d}`)
    cur.setDate(cur.getDate() + 1)
  }
  return dates
}

function buildPersonDailyMap(
  data: MultiShiftScheduleData,
): Map<string, Map<string, DayEntry[]>> {
  const shiftInfoMap = new Map<string, ShiftInfo>()
  data.shiftInfoList.forEach(info => shiftInfoMap.set(info.id, info))

  const personMap = new Map<string, Map<string, DayEntry[]>>()

  for (const [shiftId, shiftDraft] of Object.entries(data.shifts)) {
    const shiftInfo = shiftInfoMap.get(shiftId)
    const shiftName = shiftInfo?.name || `班次-${shiftId}`
    const startTime = shiftInfo?.startTime || ''

    if (!shiftDraft?.days)
      continue

    for (const [date, dayShift] of Object.entries(shiftDraft.days)) {
      const rawList = (Array.isArray(dayShift.staff) && dayShift.staff.length > 0)
        ? dayShift.staff
        : (Array.isArray(dayShift.staffIds) ? dayShift.staffIds : [])

      const staffNames = rawList
        .map((s: any) => (typeof s === 'string' ? s : String(s)))
        .filter((n: string) => n.trim().length > 0)

      for (const name of staffNames) {
        if (!personMap.has(name))
          personMap.set(name, new Map())
        const dailyMap = personMap.get(name)!
        if (!dailyMap.has(date))
          dailyMap.set(date, [])
        dailyMap.get(date)!.push({
          shiftName,
          startTime,
          isOvertime: isWeekend(date) || isNightShift(startTime),
        })
      }
    }
  }

  for (const dailyMap of personMap.values()) {
    for (const entries of dailyMap.values()) {
      entries.sort((a, b) => {
        if (a.startTime && b.startTime)
          return a.startTime.localeCompare(b.startTime)
        return 0
      })
    }
  }

  return personMap
}

export function exportScheduleByPersonToExcel(data: MultiShiftScheduleData): void {
  const dates = getDatesInRange(data.startDate, data.endDate)
  const personMap = buildPersonDailyMap(data)

  if (personMap.size === 0)
    throw new Error('暂无排班数据可导出')

  const staffOrderMap = new Map<string, { index: number, employeeId: string }>()
  if (data.staffList && data.staffList.length > 0) {
    data.staffList.forEach((emp, idx) => {
      if (emp.name)
        staffOrderMap.set(emp.name, { index: idx, employeeId: emp.employeeId || '' })
    })
  }

  const sortedNames = Array.from(personMap.keys()).sort((a, b) => {
    const infoA = staffOrderMap.get(a)
    const infoB = staffOrderMap.get(b)
    if (infoA !== undefined && infoB !== undefined) {
      const idA = infoA.employeeId
      const idB = infoB.employeeId
      const numA = (idA && !Number.isNaN(Number(idA))) ? Number(idA) : Number.POSITIVE_INFINITY
      const numB = (idB && !Number.isNaN(Number(idB))) ? Number(idB) : Number.POSITIVE_INFINITY
      if (numA !== numB)
        return numA - numB
      return idA.localeCompare(idB)
    }
    if (infoA !== undefined)
      return -1
    if (infoB !== undefined)
      return 1
    return a.localeCompare(b, 'zh')
  })

  const dateColOffset = 2
  const notesColIndex = dateColOffset + dates.length
  const overtimeColIndex = notesColIndex + 1
  const totalColIndex = notesColIndex + 2
  const totalCols = totalColIndex + 1

  const headerRow: any[] = [
    '编号',
    '姓名',
    ...dates.map(formatDateHeader),
    '备注2（更新）',
    '加班',
    '总班次',
  ]

  const sheetRows: any[][] = [headerRow]
  const merges: Array<{ s: { r: number, c: number }, e: { r: number, c: number } }> = []

  let currentRow = 1

  sortedNames.forEach((name, idx) => {
    const dailyMap = personMap.get(name)!

    let maxRows = 1
    for (const entries of dailyMap.values()) {
      if (entries.length > maxRows)
        maxRows = entries.length
    }

    let totalCount = 0
    let overtimeCount = 0
    for (const entries of dailyMap.values()) {
      for (const entry of entries) {
        totalCount++
        if (entry.isOvertime)
          overtimeCount++
      }
    }

    const rows: any[][] = Array.from(
      { length: maxRows },
      () => Array.from({ length: totalCols }).fill(''),
    )

    const empInfo = staffOrderMap.get(name)
    rows[0][0] = empInfo?.employeeId || (idx + 1)
    rows[0][1] = name
    rows[0][notesColIndex] = ''
    rows[0][overtimeColIndex] = overtimeCount
    rows[0][totalColIndex] = totalCount

    dates.forEach((date, di) => {
      const colIdx = dateColOffset + di
      const entries = dailyMap.get(date) || []
      entries.forEach((entry, ri) => {
        if (ri < maxRows)
          rows[ri][colIdx] = entry.shiftName
      })
    })

    rows.forEach(row => sheetRows.push(row))

    if (maxRows > 1) {
      const sr = currentRow
      const er = currentRow + maxRows - 1
      const mergeCols = [0, 1, notesColIndex, overtimeColIndex, totalColIndex]
      mergeCols.forEach(c =>
        merges.push({ s: { r: sr, c }, e: { r: er, c } }),
      )
    }

    currentRow += maxRows
  })

  const ws = XLSX.utils.aoa_to_sheet(sheetRows)

  if (merges.length > 0)
    ws['!merges'] = merges as XLSX.Range[]

  ws['!cols'] = [
    { wch: 6 },
    { wch: 10 },
    ...dates.map(() => ({ wch: 13 })),
    { wch: 22 },
    { wch: 6 },
    { wch: 7 },
  ]

  ws['!rows'] = [
    { hpt: 22 },
    ...Array.from({ length: sheetRows.length - 1 }, () => ({ hpt: 18 })),
  ]

  const BORDER_THIN = { style: 'thin', color: { rgb: 'AAAAAA' } } as const
  const BORDER_MEDIUM = { style: 'medium', color: { rgb: '888888' } } as const
  const BORDER_ALL_THIN = { top: BORDER_THIN, bottom: BORDER_THIN, left: BORDER_THIN, right: BORDER_THIN }
  const BORDER_ALL_MEDIUM = { top: BORDER_MEDIUM, bottom: BORDER_MEDIUM, left: BORDER_MEDIUM, right: BORDER_MEDIUM }

  const weekendColSet = new Set<number>()
  dates.forEach((date, di) => {
    if (isWeekend(date))
      weekendColSet.add(dateColOffset + di)
  })

  function makeStyle(opts: {
    isHeader?: boolean
    isWeekend?: boolean
    isIndexOrName?: boolean
    isSummary?: boolean
  }): any {
    const { isHeader, isWeekend: isWE, isIndexOrName, isSummary } = opts
    const bold = isHeader || false
    const fontSize = isHeader ? 11 : 10

    let fgColor: string
    if (isHeader)
      fgColor = 'D9E1F2'
    else if (isIndexOrName)
      fgColor = 'F2F2F2'
    else if (isSummary)
      fgColor = 'EAF0FB'
    else if (isWE)
      fgColor = 'FFF2CC'
    else
      fgColor = 'FFFFFF'

    const border = isHeader ? BORDER_ALL_MEDIUM : BORDER_ALL_THIN

    return {
      font: { name: '微软雅黑', sz: fontSize, bold },
      fill: { patternType: 'solid', fgColor: { rgb: fgColor } },
      alignment: {
        horizontal: (isHeader || isIndexOrName || isSummary) ? 'center' : 'left',
        vertical: 'center',
        wrapText: true,
      },
      border,
    }
  }

  const totalRows = sheetRows.length

  for (let r = 0; r < totalRows; r++) {
    for (let c = 0; c < totalCols; c++) {
      const addr = XLSX.utils.encode_cell({ r, c })
      if (!ws[addr])
        ws[addr] = { t: 's', v: '' }

      const isHeader = r === 0
      const isWE = weekendColSet.has(c)
      const isIndexOrName = c === 0 || c === 1
      const isSummary = c === notesColIndex || c === overtimeColIndex || c === totalColIndex

      ws[addr].s = makeStyle({ isHeader, isWeekend: isWE, isIndexOrName, isSummary })
    }
  }

  ws['!autofilter'] = { ref: XLSX.utils.encode_range({ s: { r: 0, c: 0 }, e: { r: 0, c: totalCols - 1 } }) }

  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb, ws, '排班表')
  XLSX.writeFile(wb, `排班导出_${data.startDate}_${data.endDate}.xlsx`)
}
