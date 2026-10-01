import { QRCodeSVG } from "qrcode.react";

interface QrCodeProps {
  value: string;
  /** Edge length; a CSS value such as "min(80vh, 80vw)" or pixels. */
  size: number | string;
}

/** Always black on white: an inverted code in dark mode does not scan. */
export function QrCode({ value, size }: QrCodeProps) {
  return (
    <QRCodeSVG
      value={value}
      fgColor="#000000"
      bgColor="#ffffff"
      marginSize={2}
      level="M"
      style={{ width: size, height: size, display: "block" }}
    />
  );
}
