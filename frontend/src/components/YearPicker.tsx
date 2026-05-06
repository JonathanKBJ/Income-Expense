import { DatePicker, Button } from "antd";
import { LeftOutlined, RightOutlined } from "@ant-design/icons";
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

  const handlePrev = () => {
    onYearChange(year - 1);
  };

  const handleNext = () => {
    onYearChange(year + 1);
  };

  return (
    <div className="date-navigator-wrapper" id="year-picker">
      <Button type="text" icon={<LeftOutlined />} onClick={handlePrev} className="nav-arrow-btn" />
      <DatePicker
        picker="year"
        value={value}
        onChange={handleChange}
        allowClear={false}
        format="YYYY"
        className="antd-month-picker-borderless"
        inputReadOnly
        bordered={false}
        suffixIcon={null}
      />
      <Button type="text" icon={<RightOutlined />} onClick={handleNext} className="nav-arrow-btn" />
    </div>
  );
}
