# 🧾 Receipt Reader Telegram Bot

A Telegram bot that extracts and formats data from receipt photos.

## ✨ What It Does

This bot accepts an image of a receipt and returns a neatly structured summary. It identifies items, prices, total cost, and purchase date — all in a clean, readable format. Everything is designed to work smoothly and look great in Telegram chat.

## 🔧 Features

- Accepts receipt images (photos or scans)
- Automatically detects and extracts text using OCR
- Parses and formats key information: items, prices, total
- Returns the result in a beautifully formatted message
- Handles most standard Russian receipts

## 📦 Tech Stack

- Python
- Telegram Bot API (`python-telegram-bot`)
- Tesseract OCR

## 🚀 How to Run

1. Clone the repo
2. Install dependencies
3. Set your `TELEGRAM_TOKEN`
4. Run the bot

## 🛠️ Planned Improvements

- Expense categorization
- Analytics and charts
- Export to Google Sheets
- Multilingual support
