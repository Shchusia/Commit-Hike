"""Around the World in Eighty Days (Jules Verne, 1872): a drawn route on a real world map strip.

The strip spans 384° of longitude (from 12°W eastwards to 12°E again), so London is at both
ends and the Pacific crossing doesn't break the line. Longitudes after the Pacific are +360.
"""
import json, math, os, subprocess, tempfile, zipfile, shutil

RID, OUT = "around-the-world-in-80-days", "/tmp/uploads/around-the-world-in-80-days"
LON0, LON_SPAN, LAT_TOP, LAT_SPAN = -12.0, 384.0, 72.0, 96.0
W, H = 3840, 960                       # the map picture, same 4:1 proportions
LENGTH = 800_000                        # ≈ 80 typical days of commits (10 km a day at medium)

def xy(lat, lon):
    return [round((lon - LON0) / LON_SPAN, 5), round((LAT_TOP - lat) / LAT_SPAN, 5)]

def hav(a, b):
    R = 6371000; p1, p2 = math.radians(a[0]), math.radians(b[0]); dp = p2 - p1; dl = math.radians(b[1] - a[1])
    return 2 * R * math.asin(math.sqrt(math.sin(dp / 2) ** 2 + math.cos(p1) * math.cos(p2) * math.sin(dl / 2) ** 2))

# (lat, lon unwrapped, stop or None). Stops: (id, kind, elevation, en name, en text, uk name, uk text)
S = lambda *a: a
LINE = [
 (51.507, -0.128, S("london", "start", 20, "The Reform Club, London", "Phileas Fogg bets £20,000 that he can go round the world in eighty days. He leaves the same evening.", "Реформ-клуб, Лондон", "Філеас Фоґґ ставить 20 000 фунтів, що обігне світ за вісімдесят днів. Вирушає того ж вечора.")),
 (51.12, 1.31, None), (50.95, 1.86, None),
 (48.857, 2.352, S("paris", "village", 35, "Paris", "Through Paris by night train, without stopping to look.", "Париж", "Крізь Париж нічним потягом, навіть не глянувши у вікно.")),
 (45.25, 6.90, S("mont-cenis", "pass", 1300, "Mont Cenis", "Under the Alps by the brand-new tunnel to Italy.", "Мон-Сені", "Під Альпами новеньким тунелем до Італії.")),
 (45.07, 7.69, None),
 (40.633, 17.942, S("brindisi", "harbor", 10, "Brindisi", "Aboard the steamer Mongolia for Suez.", "Бріндізі", "На пароплав «Монголія» до Суеца.")),
 (36.5, 20.5, None), (33.5, 27.0, None), (31.26, 32.30, None),
 (29.967, 32.55, S("suez", "harbor", 5, "Suez", "A detective named Fix is waiting: he is sure Fogg robbed the Bank of England.", "Суец", "Тут чекає детектив Фікс: він певен, що Фоґґ пограбував Банк Англії.")),
 (27.0, 34.3, None), (20.0, 38.8, None), (14.5, 42.3, None),
 (12.786, 45.019, S("aden", "harbor", 10, "Aden", "A coaling stop at the mouth of the Red Sea.", "Аден", "Зупинка по вугілля біля виходу з Червоного моря.")),
 (14.0, 55.0, None), (17.0, 65.0, None),
 (18.939, 72.835, S("bombay", "harbor", 10, "Bombay", "Two days ahead of schedule. Passepartout nearly loses everything at a temple.", "Бомбей", "На два дні раніше графіка. Паспарту ледь усе не втрачає біля храму.")),
 (21.15, 79.09, S("elephant", "landmark", 300, "The railway that wasn't finished", "The line stops in the jungle. Fogg buys an elephant.", "Залізниця, якої ще немає", "Колія обривається в джунглях. Фоґґ купує слона.")),
 (25.436, 81.846, S("allahabad", "village", 100, "Allahabad", "The travellers save a young widow, Aouda, from being burned against her will. She joins them.", "Аллахабад", "Мандрівники рятують молоду вдову Ауду від примусового спалення. Вона їде з ними.")),
 (25.32, 83.0, None),
 (22.573, 88.364, S("calcutta", "harbor", 10, "Calcutta", "A trial, a fine, bail paid in banknotes, and onto the next steamer.", "Калькутта", "Суд, штраф, застава банкнотами — і на наступний пароплав.")),
 (17.0, 90.5, None), (10.0, 95.5, None), (5.0, 98.5, None), (2.5, 101.5, None),
 (1.290, 103.852, S("singapore", "harbor", 10, "Singapore", "A walk among the spice gardens while the ship takes on coal.", "Сінгапур", "Прогулянка садами прянощів, поки корабель вантажить вугілля.")),
 (6.0, 106.0, None), (14.0, 112.0, None),
 (22.286, 114.158, S("hong-kong", "harbor", 10, "Hong Kong", "Fix gets Passepartout drunk, and the steamer to Yokohama leaves without Fogg.", "Гонконг", "Фікс напуває Паспарту, і пароплав до Йокогами йде без Фоґґа.")),
 (26.0, 120.0, None),
 (31.230, 121.474, S("shanghai", "harbor", 5, "Shanghai", "A tiny pilot boat, the Tankadere, races a typhoon to catch the ship for America.", "Шанхай", "Крихітний лоцманський човен «Танкадере» мчить наперегони з тайфуном, щоб устигнути на корабель до Америки.")),
 (32.5, 128.0, None), (33.5, 135.0, None),
 (35.444, 139.638, S("yokohama", "harbor", 10, "Yokohama", "Passepartout, penniless, joins a troupe of acrobats until Fogg finds him.", "Йокогама", "Паспарту без грошей пристає до трупи акробатів, поки Фоґґ його не знаходить.")),
 (38.0, 160.0, None), (39.5, 180.0, S("dateline", "landmark", 0, "The 180th meridian", "Mid-Pacific. Going east, the travellers gain a day without noticing.", "180-й меридіан", "Посеред Тихого океану. Рухаючись на схід, мандрівники непомітно виграють добу.")),
 (39.0, 200.0, None), (38.5, 222.0, None),
 (37.775, 237.58, S("san-francisco", "harbor", 15, "San Francisco", "A political rally turns into a brawl in the street.", "Сан-Франциско", "Політичний мітинг на вулиці переходить у бійку.")),
 (38.58, 238.51, None), (39.32, 239.67, None),
 (40.76, 248.11, S("salt-lake", "lake", 1300, "Great Salt Lake", "The train stops near the lake; Passepartout spends an afternoon in Salt Lake City.", "Велике Солоне озеро", "Потяг стоїть біля озера; Паспарту проводить пообідню годину в Солт-Лейк-Сіті.")),
 (41.14, 254.73, S("sherman-summit", "pass", 2440, "Sherman Summit", "The highest point of the Pacific railroad, over the Rocky Mountains.", "Перевал Шерман", "Найвища точка Тихоокеанської залізниці — над Скелястими горами.")),
 (40.65, 261.0, S("fort-kearney", "castle", 650, "Fort Kearney", "Raiders attack the train on the prairie. Passepartout is carried off, and Fogg goes after him.", "Форт-Карні", "На прерії нападники атакують потяг. Паспарту забирають у полон, і Фоґґ іде його визволяти.")),
 (41.26, 264.05, None),
 (41.878, 272.37, S("chicago", "village", 180, "Chicago", "Across the prairie by a sledge with sails, then the express east.", "Чикаго", "Через прерію на санях під вітрилом, потім експресом на схід.")),
 (40.713, 286.0, S("new-york", "harbor", 10, "New York", "The Liverpool steamer left 45 minutes ago. Fogg hires a ship and takes command of it.", "Нью-Йорк", "Пароплав до Ліверпуля пішов 45 хвилин тому. Фоґґ наймає корабель і бере команду на себе.")),
 (41.5, 300.0, None), (46.0, 320.0, S("burning-ship", "landmark", 0, "Mid-Atlantic", "Out of coal, Fogg buys the ship and burns its woodwork in the boilers.", "Посеред Атлантики", "Вугілля скінчилося — Фоґґ купує корабель і палить у топках його дерев'яні частини.")),
 (50.0, 340.0, None),
 (51.850, 351.71, S("queenstown", "harbor", 10, "Queenstown", "Ireland. A night train to Dublin and a fast ferry.", "Квінстаун", "Ірландія. Нічний потяг до Дубліна й швидкий пором.")),
 (53.35, 353.74, None),
 (53.408, 357.01, S("liverpool", "harbor", 10, "Liverpool", "On English soil at last, and Fix arrests him. The mistake costs Fogg the last hours.", "Ліверпуль", "Нарешті на англійській землі — і Фікс його заарештовує. Помилка коштує Фоґґові останніх годин.")),
 (51.507, 359.872, S("reform-club", "finish", 20, "The Reform Club again", "He thinks he has lost by five minutes. Then the day gained at the 180th meridian.", "Знову Реформ-клуб", "Він думає, що програв на п'ять хвилин. А потім — доба, виграна на 180-му меридіані.")),
]

pts = [(lat, lon) for lat, lon, _ in LINE]
cum = [0.0]
for a, b in zip(pts, pts[1:]): cum.append(cum[-1] + hav(a, b))
real_total = cum[-1]; k = LENGTH / real_total
path = [xy(lat, lon) for lat, lon in pts]
stops = [(s, round(c * k), xy(lat, lon)) for (lat, lon, s), c in zip(LINE, cum) if s]
at = {s[0]: m for s, m, _ in stops}
at["reform-club"] = LENGTH

route = {"id": RID, "version": 1, "default_locale": "en", "length_m": LENGTH, "loop": True,
         "path": path, "map_image": "assets/world-strip.svg",
         "waypoints": [{"id": s[0], "at_m": at[s[0]], "kind": s[1], "elevation_m": s[2], "x": p[0], "y": p[1]} for s, m, p in stops]}
f = lambda r: at[r] if isinstance(r, str) else round(r * LENGTH)
BIOMES = [("london", "village"), ("paris", "fields"), ("mont-cenis", "snow"), ("brindisi", "water"), ("suez", "desert"), (0.13, "water"),
          ("bombay", "village"), ("elephant", "grove"), ("calcutta", "swamp"), (0.32, "water"), ("singapore", "coast"), (0.37, "water"),
          ("hong-kong", "village"), ("shanghai", "water"), ("san-francisco", "village"), (0.69, "forest"), ("salt-lake", "steppe"),
          ("sherman-summit", "rock"), ("fort-kearney", "steppe"), ("chicago", "village"), ("new-york", "water"), ("queenstown", "meadow"), ("liverpool", "village")]
route["biomes"] = [{"at_m": f(r), "type": t} for r, t in BIOMES]
STORY = [("time", 0.05, "Fogg checks his watch at every station. He never hurries, and never waits.", "Фоґґ звіряє годинник на кожній станції. Він ніколи не поспішає й ніколи не чекає."),
         ("fix", 0.25, "The same face again on the deck. Fix is following.", "Знову те саме обличчя на палубі. Фікс іде слідом."),
         ("pacific", 0.55, "Twenty days of ocean. Passepartout's watch still shows London time.", "Двадцять днів океану. Годинник Паспарту досі показує лондонський час."),
         ("snow", 0.75, "Snow on the plains, and the train waiting for a herd of bison to cross.", "Сніг на рівнинах, і потяг чекає, поки перейде стадо бізонів.")]
FACTS = [("verne", 0.02, "Jules Verne's novel came out in 1872, first as a newspaper serial; it is in the public domain.", "Роман Жуля Верна вийшов 1872 року, спершу частинами в газеті; він у суспільному надбанні."),
         ("suez-canal", "suez", "The Suez Canal had opened in 1869, three years before the novel: it made Fogg's route possible.", "Суецький канал відкрили 1869 року, за три роки до роману, — саме він зробив маршрут Фоґґа можливим."),
         ("railroad", "san-francisco", "The first railroad across the United States was completed in May 1869.", "Першу залізницю через усі Сполучені Штати завершили в травні 1869 року."),
         ("dateline", "dateline", "Travelling east, every time zone puts the clock an hour ahead: round the whole world you gain a full day.", "Рухаючись на схід, кожен часовий пояс переводить годинник на годину вперед: за повне коло виграєш цілу добу."),
         ("bly", 0.85, "In 1889 the journalist Nellie Bly went round the world in 72 days, to beat Fogg's record.", "1889 року журналістка Неллі Блай обігнула світ за 72 дні, щоб побити рекорд Фоґґа."),
         ("scale", 0.5, "Fogg's real route is about 40,000 km; here the world is shrunk so that about 80 typical days of commits take you round.", "Справжній шлях Фоґґа — близько 40 000 км; тут світ стиснуто так, щоб близько 80 звичайних днів комітів провели тебе довкола.")]
DANGERS = [("typhoon", "shanghai", "A typhoon hits the little Tankadere. Hold on.", "Тайфун накриває маленьку «Танкадере». Тримайся."),
           ("raid", "fort-kearney", "Shots along the train. Keep your head down.", "Постріли вздовж потяга. Пригнись."),
           ("arrest", "liverpool", "Hands on your shoulder: you're under arrest.", "Рука на плечі: ви заарештовані.")]
ACH = [("atw-wager", {"type": "distance", "min_m": 2000}, ("The Wager", "Leave the Reform Club."), ("Парі", "Вийди з Реформ-клубу.")),
       ("atw-suez", {"type": "waypoint", "waypoint": "suez"}, ("Through Suez", "Reach Suez."), ("Через Суец", "Дійди до Суеца.")),
       ("atw-elephant", {"type": "waypoint", "waypoint": "elephant"}, ("By Elephant", "Cross the jungle where the railway ends."), ("На слоні", "Перетни джунглі там, де обривається колія.")),
       ("atw-dateline", {"type": "waypoint", "waypoint": "dateline"}, ("A Day Gained", "Cross the 180th meridian."), ("Виграна доба", "Перетни 180-й меридіан.")),
       ("atw-summit", {"type": "altitude", "altitude_m": 2400}, ("Over the Rockies", "Cross Sherman Summit."), ("Через Скелясті гори", "Перетни перевал Шерман.")),
       ("atw-express", {"type": "day_distance", "min_m": 20000}, ("Express Train", "Travel 20 km in one day."), ("Експрес", "Подолай 20 км за день.")),
       ("atw-round", {"type": "finish"}, ("Round the World", "Get back to the Reform Club."), ("Довкола світу", "Повернися до Реформ-клубу."))]
route["story"] = [{"id": i, "at_m": f(r)} for i, r, _, _ in STORY]
route["facts"] = [{"id": i, "at_m": f(r)} for i, r, _, _ in FACTS]
route["dangers"] = [{"id": i, "at_m": f(r)} for i, r, _, _ in DANGERS]
route["achievements"] = [{"id": i, "rule": r} for i, r, _, _ in ACH]

TEXTS = {"en": ("Around the World in Eighty Days", "Phileas Fogg's wager from Jules Verne's novel of 1872: London, Suez, Bombay, Calcutta, Hong Kong, Yokohama, San Francisco, New York and back, by steamer, train, elephant and sail-sledge. On a real map of the world, shrunk so that about 80 typical days of commits take you round."),
         "uk": ("Навколо світу за вісімдесят днів", "Парі Філеаса Фоґґа з роману Жуля Верна 1872 року: Лондон, Суец, Бомбей, Калькутта, Гонконг, Йокогама, Сан-Франциско, Нью-Йорк і назад — пароплавом, потягом, на слоні й санях під вітрилом. На справжній мапі світу, стиснутій так, щоб близько 80 звичайних днів комітів провели тебе довкола.")}

# ---------------- the map picture: Natural Earth land on an old-paper strip
land = json.load(open("/tmp/eighty/land.geojson"))
def ring(coords, shift):
    return "M" + " L".join(f"{(lon + shift - LON0) / LON_SPAN * W:.1f},{(LAT_TOP - lat) / LAT_SPAN * H:.1f}" for lon, lat in coords) + " Z"
paths = []
for feat in land["features"]:
    g = feat["geometry"]; polys = [g["coordinates"]] if g["type"] == "Polygon" else g["coordinates"]
    for poly in polys:
        for shift in (0, 360):
            paths.append(ring(poly[0], shift))
grat = []
for lon in range(-30, 400, 30):
    x = (lon - LON0) / LON_SPAN * W; grat.append(f'<line x1="{x:.0f}" y1="0" x2="{x:.0f}" y2="{H}"/>')
for lat in range(-20, 80, 20):
    y = (LAT_TOP - lat) / LAT_SPAN * H; grat.append(f'<line x1="0" y1="{y:.0f}" x2="{W}" y2="{y:.0f}"/>')
x180 = (180 - LON0) / LON_SPAN * W
svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W} {H}" width="{W}" height="{H}">
<defs><linearGradient id="sea" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#c7d3c9"/><stop offset="1" stop-color="#b3c4bd"/></linearGradient>
<radialGradient id="age" cx="0.5" cy="0.5" r="0.75"><stop offset="0.6" stop-color="#000" stop-opacity="0"/><stop offset="1" stop-color="#5b4325" stop-opacity="0.35"/></radialGradient></defs>
<rect width="{W}" height="{H}" fill="url(#sea)"/>
<g stroke="#8d9a8c" stroke-width="1.2" opacity="0.6">{"".join(grat)}</g>
<line x1="{x180:.0f}" y1="0" x2="{x180:.0f}" y2="{H}" stroke="#7c6a4d" stroke-width="3" stroke-dasharray="18,12" opacity="0.7"/>
<path d="{" ".join(paths)}" fill="#e3cf9f" stroke="#8c7449" stroke-width="2" stroke-linejoin="round"/>
<rect width="{W}" height="{H}" fill="url(#age)"/>
<rect x="6" y="6" width="{W-12}" height="{H-12}" fill="none" stroke="#6b5636" stroke-width="10"/>
<g transform="translate({W-170},{H-170})" fill="#6b5636"><circle r="70" fill="none" stroke="#6b5636" stroke-width="4"/>
<path d="M0,-90 L14,0 L0,90 L-14,0 Z"/><path d="M-90,0 L0,-10 L90,0 L0,10 Z" opacity="0.6"/></g>
</svg>'''

d = os.path.join(OUT, "pack", RID)
shutil.rmtree(os.path.join(OUT, "pack"), ignore_errors=True)
os.makedirs(os.path.join(d, "locales")); os.makedirs(os.path.join(d, "assets"))
open(os.path.join(d, "assets", "world-strip.svg"), "w").write(svg)
json.dump(route, open(os.path.join(d, "route.json"), "w"), ensure_ascii=False, indent=2)
for li, lang in enumerate(("en", "uk")):
    loc = {"name": TEXTS[lang][0], "description": TEXTS[lang][1],
           "waypoints": {s[0]: {"name": s[3 + 2 * li], "text": s[4 + 2 * li]} for s, _, _ in stops},
           "story": {i: (en, uk)[li] for i, _, en, uk in STORY}, "facts": {i: (en, uk)[li] for i, _, en, uk in FACTS},
           "dangers": {i: (en, uk)[li] for i, _, en, uk in DANGERS},
           "achievements": {i: {"name": (en, uk)[li][0], "description": (en, uk)[li][1]} for i, _, en, uk in ACH}}
    json.dump(loc, open(os.path.join(d, "locales", f"{lang}.json"), "w"), ensure_ascii=False, indent=2)
zpath = os.path.join(OUT, f"{RID}.zip")
with zipfile.ZipFile(zpath, "w", zipfile.ZIP_DEFLATED) as z:
    for root, _, files in os.walk(d):
        for fn in files:
            p = os.path.join(root, fn); z.write(p, os.path.relpath(p, os.path.dirname(d)))
with tempfile.TemporaryDirectory() as home:
    res = json.loads(subprocess.run(["/tmp/chs", "route", "import", "--path", zpath], env={"COMMIT_HIKE_HOME": home}, capture_output=True, text=True).stdout)
if not res.get("ok"): raise SystemExit(res["error"]["message"])
print(f"valid: {LENGTH/1000:.0f} km (real route {real_total/1000:,.0f} km, ×{1/k:.0f}), {len(stops)} stops, {len(FACTS)} facts, "
      f"{len(DANGERS)} dangers, map {len(svg)//1024} KB, zip {os.path.getsize(zpath)//1024} KB")
open(os.path.join(OUT, "world-strip.svg"), "w").write(svg)
