import { DatePicker, Button } from "antd";
import { LeftOutlined, RightOutlined } from "@ant-design/icons";
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
  const value = dayjs().year(year).month(month - 1);

  const handleChange = (date: dayjs.Dayjs | null) => {
    if (date) {
      onMonthChange(date.month() + 1);
      onYearChange(date.year());
    }
  };

  const handlePrev = () => {
    const newDate = value.subtract(1, 'month');
    onMonthChange(newDate.month() + 1);
    onYearChange(newDate.year());
  };

  const handleNext = () => {
    const newDate = value.add(1, 'month');
    onMonthChange(newDate.month() + 1);
    onYearChange(newDate.year());
  };

  return (
    <div className="date-navigator-wrapper" id="month-picker">
      <Button type="text" icon={<LeftOutlined />} onClick={handlePrev} className="nav-arrow-btn" />
      <DatePicker
        picker="month"
        value={value}
        onChange={handleChange}
        allowClear={false}
        format="MMM YYYY"
        className="antd-month-picker-borderless"
        inputReadOnly
        bordered={false}
        suffixIcon={null}
      />
      <Button type="text" icon={<RightOutlined />} onClick={handleNext} className="nav-arrow-btn" />
    </div>
  );
}
