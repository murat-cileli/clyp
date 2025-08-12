#!/bin/bash

# GTK4 Go Uygulaması Tema Test Scripti
# Bu script, uygulamanın GNOME tema desteğini test eder

echo "🎨 GTK4 Go Uygulaması Tema Test Scripti"
echo "========================================"

# Mevcut tema ayarını kontrol et
echo "📋 Mevcut tema ayarları:"
echo "  Color Scheme: $(gsettings get org.gnome.desktop.interface color-scheme)"
echo "  GTK Theme: $(gsettings get org.gnome.desktop.interface gtk-theme)"

echo ""
echo "🔄 Tema değişiklik testi başlıyor..."

# Karanlık temaya geç
echo "🌙 Karanlık temaya geçiliyor..."
gsettings set org.gnome.desktop.interface color-scheme 'prefer-dark'
echo "  ✅ Karanlık tema ayarlandı"
sleep 3

# Açık temaya geç
echo "☀️  Açık temaya geçiliyor..."
gsettings set org.gnome.desktop.interface color-scheme 'default'
echo "  ✅ Açık tema ayarlandı"
sleep 3

# Tekrar karanlık temaya geç
echo "🌙 Tekrar karanlık temaya geçiliyor..."
gsettings set org.gnome.desktop.interface color-scheme 'prefer-dark'
echo "  ✅ Karanlık tema ayarlandı"
sleep 3

# Son olarak açık temaya dön
echo "☀️  Son olarak açık temaya dönülüyor..."
gsettings set org.gnome.desktop.interface color-scheme 'default'
echo "  ✅ Açık tema ayarlandı"

echo ""
echo "✨ Tema testi tamamlandı!"
echo "📝 Uygulama penceresinin tema değişikliklerini takip ettiğini gözlemleyin."

echo ""
echo "🔧 Manuel test komutları:"
echo "  Karanlık tema: gsettings set org.gnome.desktop.interface color-scheme 'prefer-dark'"
echo "  Açık tema:    gsettings set org.gnome.desktop.interface color-scheme 'default'"
echo "  Mevcut ayar:  gsettings get org.gnome.desktop.interface color-scheme"
