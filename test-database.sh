#!/bin/bash

# GTK4 Go Uygulaması SQLite Veritabanı Test Scripti
# Bu script, uygulamanın SQLite entegrasyonunu test eder

echo "🗄️  GTK4 Go Uygulaması SQLite Test Scripti"
echo "==========================================="

# Veritabanı dosyasının varlığını kontrol et
if [ -f "clipboard.db" ]; then
    echo "✅ clipboard.db dosyası mevcut"
else
    echo "❌ clipboard.db dosyası bulunamadı"
    exit 1
fi

echo ""
echo "📊 Veritabanı bilgileri:"
echo "  Dosya boyutu: $(du -h clipboard.db | cut -f1)"
echo "  Tablo yapısı:"
sqlite3 clipboard.db ".schema clipboard"

echo ""
echo "📝 Mevcut veriler (son 5 kayıt):"
sqlite3 clipboard.db "SELECT content FROM clipboard WHERE content IS NOT NULL AND content != '' ORDER BY date_time DESC LIMIT 5;"

echo ""
echo "🧪 Test verisi ekleniyor..."
sqlite3 clipboard.db "INSERT INTO clipboard (content) VALUES ('Test scripti ile eklenen veri - $(date)');"

echo "✅ Test verisi eklendi"

echo ""
echo "📊 Toplam kayıt sayısı:"
sqlite3 clipboard.db "SELECT COUNT(*) FROM clipboard WHERE content IS NOT NULL AND content != '';"

echo ""
echo "🔄 Uygulamayı yeniden başlatarak yeni verileri görebilirsiniz."
echo ""
echo "💡 Kullanışlı komutlar:"
echo "  Tüm verileri göster: sqlite3 clipboard.db \"SELECT * FROM clipboard ORDER BY date_time DESC;\""
echo "  Veri ekle:           sqlite3 clipboard.db \"INSERT INTO clipboard (content) VALUES ('Yeni veri');\""
echo "  Veri sil:            sqlite3 clipboard.db \"DELETE FROM clipboard WHERE id = [ID];\""
