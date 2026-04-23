import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import ThemeProvider from "@/components/ui/theme-provider";
import ToastContainer from "@/components/ui/toast";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: {
    default: "Saajan - Fashion & Lifestyle",
    template: "%s | Saajan",
  },
  description: "Premium fashion and lifestyle products from Bangladesh. Shop sarees, panjabis, accessories, and more.",
  keywords: ["saajan", "fashion", "bangladesh", "ecommerce", "saree", "panjabi", "online shopping"],
  openGraph: {
    type: "website",
    locale: "en_BD",
    siteName: "Saajan",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${geistSans.variable} ${geistMono.variable}`} suppressHydrationWarning>
      <body className="min-h-screen antialiased bg-surface-secondary text-text">
        <ThemeProvider>{children}</ThemeProvider>
        <ToastContainer />
      </body>
    </html>
  );
}
