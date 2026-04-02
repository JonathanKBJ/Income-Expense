import { DatePicker } from "antd";
import dayjs from "dayjs";

interface MonthPickerProps {
  month: number;
  year: number;
  onMonthChange: (m: number) => void;
  onYearChange: (y: number) => void;
}

export default function MonthPicker({
  month,
  year,
  onMonthChange,
  onYearChange,
}: MonthPickerProps) {
  // Create a dayjs object from month/year props
  // dayjs months are 0-indexed, but our props are 1-indexed
  const value = dayjs().year(year).month(month - 1);

  const handleChange = (date: dayjs.Dayjs | null) => {
    if (date) {
      onMonthChange(date.month() + 1);
      onYearChange(date.year());
    }
  };

  return (
    <div className="month-picker-wrapper" id="month-picker">
      <DatePicker
        picker="month"
        value={value}
        onChange={handleChange}
        allowClear={false}
        format="MMM YYYY"
        className="antd-month-picker"
        inputReadOnly
      />
    </div>
  );
}
