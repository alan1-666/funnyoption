import type { Metadata } from "next";
import { Anton, IBM_Plex_Mono, IBM_Plex_Sans, Instrument_Serif } from "next/font/google";

import "@/app/globals.css";
import { Providers } from "@/app/providers";

const display = Anton({
  subsets: ["latin"],
  variable: "--font-display",
  weight: "400"
});

const body = IBM_Plex_Sans({
  subsets: ["latin"],
  variable: "--font-body",
  weight: ["400", "500", "600", "700"]
});

const mono = IBM_Plex_Mono({
  subsets: ["latin"],
  variable: "--font-mono",
  weight: ["400", "500"]
});

const serif = Instrument_Serif({
  subsets: ["latin"],
  variable: "--font-serif",
  weight: "400"
});

export const metadata: Metadata = {
  title: "FunnyOption",
  description: "FunnyOption 中文交易前端，提供本地市场浏览、充值、下单和结算查看。"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body className={`${display.variable} ${body.variable} ${mono.variable} ${serif.variable}`}>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
