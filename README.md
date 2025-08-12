# GTK4 Go Uygulaması - SQLite & GNOME Tema Desteği

Bu uygulama, SQLite veritabanı entegrasyonu ve GNOME'un karanlık/açık tema tercihini otomatik olarak destekleyen GTK4 Go uygulamasıdır. Uygulama, modern bir arayüz ile HeaderBar, ListBox ve menü desteği içerir.

## Özellikler

- ✅ **SQLite Veritabanı Entegrasyonu**: `clipboard.db` dosyasından veri okuma
- ✅ **Tarih/Saat Desteği**: `date_time` kolonunu alt başlık olarak gösterme
- ✅ **Akıllı Tarih Formatı**: "2 saat önce", "3 gün önce" gibi kullanıcı dostu format
- ✅ **Clipboard Kopyalama**: Çift tıklama veya Enter ile panoya kopyalama
- ✅ **Wayland Uyumluluğu**: Linux Wayland ile tam uyumlu clipboard desteği
- ✅ **Kaydırılabilir Liste**: ScrolledWindow ile büyük veri setleri için optimizasyon
- ✅ **Modern CSS Stilleri**: Özel CSS ile gelişmiş görsel tasarım
- ✅ **GNOME Tema Desteği**: Sistem tema tercihini otomatik algılama
- ✅ **Klavye Desteği**: Enter tuşu ile hızlı kopyalama

## Nasıl Çalışır

Uygulama aşağıdaki yöntemlerle SQLite entegrasyonu ve GNOME tema desteği sağlar:

### SQLite Veritabanı Entegrasyonu
1. **Veritabanı Bağlantısı**: Uygulama başladığında `./clipboard.db` dosyasına bağlanır
2. **Veri Okuma**: `clipboard` tablosundaki `content` ve `date_time` kolonlarından verileri okur
3. **Filtreleme**: Sadece anlamlı metinsel içerikleri gösterir (sayısal veriler filtrelenir)
4. **Sıralama**: Veriler tarih sırasına göre (en yeni önce) listelenir
5. **Tarih Formatı**: Akıllı tarih gösterimi ("2 saat önce", "3 gün önce", "15.12.2024 14:30")
6. **Kaydırılabilir Arayüz**: ScrolledWindow ile 100+ kayıt için optimize edilmiş görüntüleme

### GNOME Tema Desteği
1. **GSettings Entegrasyonu**: `org.gnome.desktop.interface` şemasından `color-scheme` ayarını okur
2. **Gerçek Zamanlı Takip**: Sistem tema değişikliklerini dinler ve otomatik olarak günceller
3. **GTK4 Uyumluluğu**: `gtk-application-prefer-dark-theme` özelliğini kullanarak GTK4'ün yerli tema desteğinden yararlanır

### Clipboard Kopyalama Özelliği
1. **Çift Tıklama**: Herhangi bir liste öğesine çift tıklayarak içeriği panoya kopyalayın
2. **Enter Tuşu**: Bir öğeyi seçip Enter tuşuna basarak kopyalayın
3. **Wayland Desteği**: Linux Wayland ortamında tam uyumlu çalışır
4. **Sadece İçerik**: Tarih bilgisi hariç, sadece ana içerik kopyalanır

## Derleme ve Çalıştırma

```bash
# Bağımlılıkları yükle
go mod tidy

# Derle
go build -o go-gtk4-app .

# Çalıştır
./go-gtk4-app
```

## Tema Testi

Uygulamanın tema desteğini test etmek için otomatik test scripti kullanabilirsiniz:

```bash
# Otomatik tema testi
./test-theme.sh
```

Veya manuel olarak:

```bash
# Karanlık temaya geç
gsettings set org.gnome.desktop.interface color-scheme 'prefer-dark'

# Açık temaya geç
gsettings set org.gnome.desktop.interface color-scheme 'default'

# Mevcut ayarı kontrol et
gsettings get org.gnome.desktop.interface color-scheme
```

## Teknik Detaylar

### Kullanılan Kütüphaneler
- `github.com/diamondburned/gotk4/pkg/gtk/v4` - GTK4 bağlamaları
- `github.com/diamondburned/gotk4/pkg/gdk/v4` - GDK4 (CSS ve display desteği)
- `github.com/diamondburned/gotk4/pkg/gio/v2` - GIO/GSettings desteği
- `github.com/mattn/go-sqlite3` - SQLite veritabanı desteği

### Tema Algılama Mantığı
1. Uygulama başladığında `org.gnome.desktop.interface` şemasından `color-scheme` değeri okunur
2. Bu değer `prefer-dark` ise `gtk-application-prefer-dark-theme` true olarak ayarlanır
3. GSettings değişiklik sinyalleri dinlenerek tema değişiklikleri gerçek zamanlı olarak takip edilir

### Fallback Mekanizması
GNOME ayarları mevcut değilse, uygulama GTK tema adından çıkarım yapar:
- `Adwaita-dark` → Karanlık tema
- `Adwaita` → Açık tema

## SQLite Veritabanı Kullanımı

### Veritabanı Yapısı
Uygulama `./clipboard.db` dosyasındaki `clipboard` tablosunu kullanır:

```sql
CREATE TABLE clipboard (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content TEXT NOT NULL,
    date_time TEXT DEFAULT CURRENT_TIMESTAMP,
    type INTEGER DEFAULT 1,
    is_pinned INTEGER DEFAULT 0,
    is_encrypted INTEGER DEFAULT 0
);
```

### Veri Ekleme
Veritabanına yeni clipboard verisi eklemek için:

```bash
# SQLite komut satırı ile
sqlite3 clipboard.db "INSERT INTO clipboard (content) VALUES ('Yeni clipboard verisi');"

# Birden fazla veri eklemek için
sqlite3 clipboard.db "INSERT INTO clipboard (content) VALUES
    ('İlk metin'),
    ('İkinci metin'),
    ('Üçüncü metin');"
```

### Veri Görüntüleme
Mevcut verileri görüntülemek için:

```bash
# Tüm verileri göster
sqlite3 clipboard.db "SELECT * FROM clipboard ORDER BY date_time DESC;"

# Sadece content kolonunu göster
sqlite3 clipboard.db "SELECT content FROM clipboard ORDER BY date_time DESC LIMIT 10;"
```

## Sorun Giderme

### SQLite Veritabanı Sorunları
Eğer veritabanı dosyası bulunamazsa veya bozuksa:

```bash
# Veritabanı dosyasını kontrol et
ls -la clipboard.db

# Veritabanı yapısını kontrol et
sqlite3 clipboard.db ".schema clipboard"

# Yeni veritabanı oluştur (gerekirse)
sqlite3 clipboard.db "CREATE TABLE clipboard (id INTEGER PRIMARY KEY AUTOINCREMENT, content TEXT NOT NULL, date_time TEXT DEFAULT CURRENT_TIMESTAMP);"
```

### GNOME Ayarları Bulunamıyor
Eğer `org.gnome.desktop.interface` şeması bulunamazsa:
```bash
# Şemaların listesini kontrol et
gsettings list-schemas | grep desktop.interface

# GNOME ayarlarını sıfırla
gsettings reset org.gnome.desktop.interface color-scheme
```

### GTK Tema Ayarları
Manuel GTK tema ayarları için:
```bash
# GTK ayarlarını kontrol et
gsettings list-keys org.gnome.desktop.interface

# Tema adını kontrol et
gsettings get org.gnome.desktop.interface gtk-theme
```

## 🖱️ Kullanım Kılavuzu

### Temel Kullanım

1. **Uygulamayı Başlatın**:
   ```bash
   ./go-gtk4-app
   ```

2. **Clipboard Verilerini Görüntüleme**:
   - Uygulama açıldığında SQLite veritabanından veriler otomatik yüklenir
   - Her öğe ana içerik ve tarih bilgisi ile gösterilir

3. **Panoya Kopyalama**:
   - **Çift Tıklama**: Herhangi bir liste öğesine çift tıklayın
   - **Enter Tuşu**: Bir öğeyi seçip Enter tuşuna basın
   - Sadece ana içerik kopyalanır (tarih bilgisi hariç)

4. **Yapıştırma**:
   - Başka bir uygulamada `Ctrl+V` ile yapıştırın
   - Wayland ve X11 ortamlarında çalışır

### Test Etme

Clipboard kopyalama özelliğini test etmek için:

```bash
# Terminal'de uygulamayı çalıştırın
./go-gtk4-app

# Başka bir terminal açın ve şunu çalıştırın:
xclip -o        # X11 için
# veya
wl-paste        # Wayland için
# veya
xsel --clipboard --output  # Alternatif X11 komutu
```

### Klavye Kısayolları

- **Enter**: Seçili öğeyi panoya kopyala
- **↑/↓**: Liste öğeleri arasında gezin
- **Tab**: Arayüz öğeleri arasında geçiş

## Lisans

Bu proje MIT lisansı altında lisanslanmıştır.
