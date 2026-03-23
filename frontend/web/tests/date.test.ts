/**
 * 日期工具函数测试示例
 *
 * 运行此文件可以查看生成的提示词效果
 */

import {
  formatDate,
  formatDateChinese,
  generateNextWeekSchedulePrompt,
  generateThisMonthSchedulePrompt,
  generateThisWeekSchedulePrompt,
  getNextWeekRange,
  getThisMonthRange,
  getThisWeekRange,
} from './date'

console.log('=== 日期范围测试 ===')
console.log('本周:', getThisWeekRange())
console.log('下周:', getNextWeekRange())
console.log('本月:', getThisMonthRange())

console.log('\n=== 日期格式化测试 ===')
const today = new Date()
console.log('标准格式:', formatDate(today))
console.log('中文格式:', formatDateChinese(today))

console.log('\n=== 提示词生成测试 ===')
console.log('本周排班提示词:')
console.log(generateThisWeekSchedulePrompt())
console.log('\n带部门:')
console.log(generateThisWeekSchedulePrompt('放射科'))
console.log('\n带部门和检查类型:')
console.log(generateThisWeekSchedulePrompt('放射科', 'CT'))

console.log('\n下周排班提示词:')
console.log(generateNextWeekSchedulePrompt())

console.log('\n本月排班提示词:')
console.log(generateThisMonthSchedulePrompt())
