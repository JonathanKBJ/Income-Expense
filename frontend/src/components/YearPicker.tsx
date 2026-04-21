import { DatePicker } from "antd";
import dayjs from "dayjs";

interface YearPickerProps {
  year: number;
  onYearChange: (y: number) => void;
}

export default function YearPicker({ year, onYearChange }: YearPickerProps) {
  const value = dayjs().year(year);

  const handleChange = (date: dayjs.Dayjs | null) => {
    if (date) {
      onYearChange(date.year());
    }
  };

  return (
    <div className="month-picker-wrapper" id="year-picker">
      <DatePicker
        picker="year"
        value={value}
        onChange={handleChange}
        allowClear={false}
        format="YYYY"
        className="antd-month-picker"
        inputReadOnly
      />
    </div>
  );
}
