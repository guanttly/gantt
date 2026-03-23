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
  startTime?: string // "HH:MM" 开始时间，用于判断夜班
  endTime?: string // "HH:MM" 结束时间
  isOvernight?: boolean
  type?: string
}

interface StaffInfo {
  id: string
  name: string
  employeeId?: string // 工号，用于排序
}

interface MultiShiftScheduleData {
  startDate: string
  endDate: string
  shifts: Record<string, ShiftDraft>
  shiftInfoList: ShiftInfo[]
  staffList?: StaffInfo[] // 有序员工列表，用于导出排序
}

/** 某一天某个班次的条目 */
interface DayEntry {
  shiftName: string
  startTime: string
  isOvertime: boolean
}

// ============================================================
// 工具函数
// ============================================================

/** 解析日期字符串为本地时间 Date */
function parseDateLocal(dateStr: string): Date {
  const [year, month, day] = dateStr.split('-').map(Number)
  return new Date(year, month - 1, day)
}

/** 格式化日期为表头格式 "MM.DD周X" */
function formatDateHeader(dateStr: string): string {
  const date = parseDateLocal(dateStr)
  const [, month, day] = dateStr.split('-').map(Number)
  const WEEKDAY = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return `${String(month).padStart(2, '0')}.${String(day).padStart(2, '0')}${WEEKDAY[date.getDay()]}`
}

/**
 * 判断是否为夜班：开始时间 >= 18:00
 * 夜班（晚上班次）算加班
 */
function isNightShift(startTime: string | undefined): boolean {
  if (!startTime)
    return false
  const [h] = startTime.split(':').map(Number)
  return h >= 18
}

/**
 * 判断是否为周末（周六或周日）
 * 周末班次算加班
 */
function isWeekend(dateStr: string): boolean {
  const wd = parseDateLocal(dateStr).getDay()
  return wd === 0 || wd === 6
}

/** 生成日期范围内的所有日期（含首尾） */
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

// ============================================================
// 数据转换：按人员维度汇总排班
// ============================================================

/**
 * 构建 人员姓名 → 日期 → 班次条目列表 的映射
 */
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
      // 优先用 staff（姓名），没有再用 staffIds
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
          // 加班：周末班次（周六/周日） OR 夜班（开始时间 >= 18:00）
          isOvertime: isWeekend(date) || isNightShift(startTime),
        })
      }
    }
  }

  // 同一天内的班次按开始时间排序
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

// ============================================================
// 主导出函数
// ============================================================

/**
 * 按人员维度导出排班为 Excel
 *
 * 格式（对应图示）：
 *   编号 | 姓名 | MM.DD周X | ... | 备注2（更新）| 加班 | 总班次
 *
 * - 每人可占多行（取决于某天最多同时排几个班次）
 * - 编号/姓名/备注2/加班/总班次列在多行间合并单元格
 * - 加班 = 周六/周日班次 + 夜班（开始时间 >= 18:00）的总次数
 * - 总班次 = 该周期内所有班次总数
 */
export function exportScheduleByPersonToExcel(data: MultiShiftScheduleData): void {
  const dates = getDatesInRange(data.startDate, data.endDate)
  const personMap = buildPersonDailyMap(data)

  if (personMap.size === 0) {
    throw new Error('暂无排班数据可导出')
  }

  // 按工号（employeeId）排序：优先用 staffList 顺序，其次按工号字符串升序，最后按姓名兜底
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
    // 两者都在 staffList 中 → 按员工编号数值升序
    if (infoA !== undefined && infoB !== undefined) {
      const idA = infoA.employeeId
      const idB = infoB.employeeId
      // 空工号或非数字工号排到末尾（Number('') === 0 会导致误排，需显式排除）
      const numA = (idA && !Number.isNaN(Number(idA))) ? Number(idA) : Number.POSITIVE_INFINITY
      const numB = (idB && !Number.isNaN(Number(idB))) ? Number(idB) : Number.POSITIVE_INFINITY
      if (numA !== numB)
        return numA - numB
      // 都是 Infinity（非纯数字工号）则按字符串升序
      return idA.localeCompare(idB)
    }
    // 只有一方在 staffList 中 → 在 staffList 中的排前面
    if (infoA !== undefined)
      return -1
    if (infoB !== undefined)
      return 1
    // 都不在 staffList 中 → 按姓名排序
    return a.localeCompare(b, 'zh')
  })

  // -------- 列索引 --------
  const dateColOffset = 2 // 编号(0) + 姓名(1)
  const notesColIndex = dateColOffset + dates.length
  const overtimeColIndex = notesColIndex + 1
  const totalColIndex = notesColIndex + 2
  const totalCols = totalColIndex + 1

  // -------- 表头行 --------
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

  let currentRow = 1 // 0-indexed，第 0 行是表头

  sortedNames.forEach((name, idx) => {
    const dailyMap = personMap.get(name)!

    // 该人员某天最多同时有几个班次 → 决定占几行
    let maxRows = 1
    for (const entries of dailyMap.values()) {
      if (entries.length > maxRows)
        maxRows = entries.length
    }

    // 统计加班次数和总班次
    let totalCount = 0
    let overtimeCount = 0
    for (const entries of dailyMap.values()) {
      for (const entry of entries) {
        totalCount++
        if (entry.isOvertime)
          overtimeCount++
      }
    }

    // 初始化该人员的所有行（全部填空）
    const rows: any[][] = Array.from(
      { length: maxRows },
      () => Array.from({ length: totalCols }).fill(''),
    )

    // 第一行填编号、姓名、合并列的值
    // 编号优先使用员工工号（employeeId），没有则降级为序号
    const empInfo = staffOrderMap.get(name)
    rows[0][0] = empInfo?.employeeId || (idx + 1)
    rows[0][1] = name
    rows[0][notesColIndex] = ''
    rows[0][overtimeColIndex] = overtimeCount
    rows[0][totalColIndex] = totalCount

    // 按日期填班次（同一天多个班次分散到不同子行）
    dates.forEach((date, di) => {
      const colIdx = dateColOffset + di
      const entries = dailyMap.get(date) || []
      entries.forEach((entry, ri) => {
        if (ri < maxRows) {
          rows[ri][colIdx] = entry.shiftName
        }
      })
    })

    // 把该人员的行追加到 sheet
    rows.forEach(row => sheetRows.push(row))

    // 多行时：合并 编号/姓名/备注2/加班/总班次 列
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

  // -------- 生成工作表 --------
  const ws = XLSX.utils.aoa_to_sheet(sheetRows)

  if (merges.length > 0) {
    ws['!merges'] = merges as XLSX.Range[]
  }

  // 列宽
  ws['!cols'] = [
    { wch: 6 }, // 编号
    { wch: 10 }, // 姓名
    ...dates.map(() => ({ wch: 13 })), // 日期列
    { wch: 22 }, // 备注2
    { wch: 6 }, // 加班
    { wch: 7 }, // 总班次
  ]

  // 行高：表头稍高，数据行固定
  ws['!rows'] = [
    { hpt: 22 }, // 表头行
    ...Array.from({ length: sheetRows.length - 1 }, () => ({ hpt: 18 })),
  ]

  // -------- 样式定义 --------
  const BORDER_THIN = { style: 'thin', color: { rgb: 'AAAAAA' } } as const
  const BORDER_MEDIUM = { style: 'medium', color: { rgb: '888888' } } as const
  const BORDER_ALL_THIN = { top: BORDER_THIN, bottom: BORDER_THIN, left: BORDER_THIN, right: BORDER_THIN }
  const BORDER_ALL_MEDIUM = { top: BORDER_MEDIUM, bottom: BORDER_MEDIUM, left: BORDER_MEDIUM, right: BORDER_MEDIUM }

  // 判断某列是否是周末日期列
  const weekendColSet = new Set<number>()
  dates.forEach((date, di) => {
    if (isWeekend(date))
      weekendColSet.add(dateColOffset + di)
  })

  /** 生成单元格样式 */
  function makeStyle(opts: {
    isHeader?: boolean
    isWeekend?: boolean
    isIndexOrName?: boolean // 编号/姓名列
    isSummary?: boolean // 加班/总班次/备注列
    isMergeTop?: boolean // 合并区域首行
    isMergeBody?: boolean // 合并区域非首行
    isEmpty?: boolean // 空值单元格（占位行）
  }): any {
    const { isHeader, isWeekend: isWE, isIndexOrName, isSummary } = opts

    const bold = isHeader || false
    const fontSize = isHeader ? 11 : 10

    // 背景色
    let fgColor: string
    if (isHeader) {
      fgColor = 'D9E1F2' // 浅蓝灰 - 表头
    }
    else if (isIndexOrName) {
      fgColor = 'F2F2F2' // 浅灰 - 编号/姓名
    }
    else if (isSummary) {
      fgColor = 'EAF0FB' // 淡蓝 - 统计列
    }
    else if (isWE) {
      fgColor = 'FFF2CC' // 浅黄 - 周末列
    }
    else {
      fgColor = 'FFFFFF' // 白色 - 普通日期列
    }

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

  // -------- 逐格应用样式 --------
  const totalRows = sheetRows.length

  for (let r = 0; r < totalRows; r++) {
    for (let c = 0; c < totalCols; c++) {
      const addr = XLSX.utils.encode_cell({ r, c })
      // aoa_to_sheet 对空字符串不创建单元格，需要补全
      if (!ws[addr]) {
        ws[addr] = { t: 's', v: '' }
      }

      const isHeader = r === 0
      const isWE = weekendColSet.has(c)
      const isIndexOrName = c === 0 || c === 1
      const isSummary = c === notesColIndex || c === overtimeColIndex || c === totalColIndex

      ws[addr].s = makeStyle({ isHeader, isWeekend: isWE, isIndexOrName, isSummary })
    }
  }

  // -------- 开启自动筛选（表头行） --------
  ws['!autofilter'] = { ref: XLSX.utils.encode_range({ s: { r: 0, c: 0 }, e: { r: 0, c: totalCols - 1 } }) }

  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb, ws, '排班表')
  XLSX.writeFile(wb, `排班导出_${data.startDate}_${data.endDate}.xlsx`)
}
