#!/bin/bash

# GTK4 Go Uygulaması Clipboard Test Scripti
# Bu script, uygulamanın clipboard kopyalama özelliğini test eder

echo "📋 GTK4 Go Uygulaması Clipboard Test Scripti"
echo "============================================="

# Gerekli araçların varlığını kontrol et
echo ""
echo "🔧 Gerekli araçlar kontrol ediliyor..."

# X11 için xclip kontrolü
if command -v xclip &> /dev/null; then
    echo "✅ xclip mevcut (X11 desteği)"
    X11_AVAILABLE=true
else
    echo "❌ xclip bulunamadı (X11 desteği yok)"
    X11_AVAILABLE=false
fi

# Wayland için wl-paste kontrolü
if command -v wl-paste &> /dev/null; then
    echo "✅ wl-paste mevcut (Wayland desteği)"
    WAYLAND_AVAILABLE=true
else
    echo "❌ wl-paste bulunamadı (Wayland desteği yok)"
    WAYLAND_AVAILABLE=false
fi

# Alternatif X11 aracı xsel kontrolü
if command -v xsel &> /dev/null; then
    echo "✅ xsel mevcut (X11 alternatif)"
    XSEL_AVAILABLE=true
else
    echo "❌ xsel bulunamadı"
    XSEL_AVAILABLE=false
fi

echo ""
echo "🖥️  Mevcut ortam:"
if [ -n "$WAYLAND_DISPLAY" ]; then
    echo "  Wayland ortamı tespit edildi"
    CURRENT_ENV="wayland"
elif [ -n "$DISPLAY" ]; then
    echo "  X11 ortamı tespit edildi"
    CURRENT_ENV="x11"
else
    echo "  Grafik ortamı tespit edilemedi"
    CURRENT_ENV="unknown"
fi

echo ""
echo "📝 Test veritabanına örnek veri ekleniyor..."
sqlite3 clipboard.db "INSERT INTO clipboard (content) VALUES ('Test verisi - Clipboard kopyalama testi $(date)');"

echo ""
echo "🚀 Test talimatları:"
echo "1. Uygulamayı başlatın: ./go-gtk4-app"
echo "2. Listeden bir öğeye çift tıklayın veya seçip Enter'a basın"
echo "3. Bu scripti tekrar çalıştırarak clipboard içeriğini kontrol edin"

echo ""
echo "📋 Mevcut clipboard içeriği:"

# Clipboard içeriğini oku
if [ "$CURRENT_ENV" = "wayland" ] && [ "$WAYLAND_AVAILABLE" = true ]; then
    echo "  Wayland clipboard içeriği:"
    wl-paste 2>/dev/null || echo "  (Clipboard boş veya okunamıyor)"
elif [ "$CURRENT_ENV" = "x11" ] && [ "$X11_AVAILABLE" = true ]; then
    echo "  X11 clipboard içeriği (xclip):"
    xclip -o -selection clipboard 2>/dev/null || echo "  (Clipboard boş veya okunamıyor)"
elif [ "$XSEL_AVAILABLE" = true ]; then
    echo "  X11 clipboard içeriği (xsel):"
    xsel --clipboard --output 2>/dev/null || echo "  (Clipboard boş veya okunamıyor)"
else
    echo "  ❌ Clipboard okuma aracı bulunamadı"
fi

echo ""
echo "💡 Kullanışlı komutlar:"
if [ "$WAYLAND_AVAILABLE" = true ]; then
    echo "  Wayland clipboard oku: wl-paste"
fi
if [ "$X11_AVAILABLE" = true ]; then
    echo "  X11 clipboard oku:    xclip -o -selection clipboard"
fi
if [ "$XSEL_AVAILABLE" = true ]; then
    echo "  X11 clipboard oku:    xsel --clipboard --output"
fi

echo ""
echo "🔄 Test döngüsü:"
echo "  1. Uygulamada bir öğeyi kopyalayın"
echo "  2. Bu scripti tekrar çalıştırın: ./test-clipboard.sh"
echo "  3. Clipboard içeriğinin değişip değişmediğini kontrol edin"
