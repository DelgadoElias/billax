/**
 * Format currency amount (in centavos) to string
 * @param amount Amount in centavos (e.g., 10050 = $100.50 ARS)
 * @param currency ISO 4217 currency code (default: ARS)
 * @returns Formatted currency string
 */
export function formatCurrency(
  amount: number | null | undefined,
  currency: string = 'ARS'
): string {
  if (amount === null || amount === undefined) {
    return '-'
  }

  const majorUnits = amount / 100
  const formatter = new Intl.NumberFormat('es-AR', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })

  return formatter.format(majorUnits)
}

/**
 * Format date to locale string
 * @param dateStr ISO 8601 date string or Date object
 * @param format 'short' | 'long' | 'time' (default: short)
 * @returns Formatted date string
 */
export function formatDate(
  dateStr: string | Date | null | undefined,
  format: 'short' | 'long' | 'time' = 'short'
): string {
  if (!dateStr) {
    return '-'
  }

  const date = typeof dateStr === 'string' ? new Date(dateStr) : dateStr

  if (isNaN(date.getTime())) {
    return '-'
  }

  const options: Intl.DateTimeFormatOptions =
    format === 'time'
      ? {
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit',
        }
      : format === 'long'
        ? {
            weekday: 'long',
            year: 'numeric',
            month: 'long',
            day: 'numeric',
          }
        : {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
          }

  return date.toLocaleDateString('es-AR', options)
}

/**
 * Truncate text to max length with ellipsis
 */
export function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) {
    return text
  }
  return text.substring(0, maxLength - 3) + '...'
}
