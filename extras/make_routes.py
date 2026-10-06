import json, math, os, io, zipfile, subprocess, shutil, tempfile
OUT = '/tmp/uploads'
CORE = '/tmp/chs'

def hav(a, b):
    R = 6371000; p1, p2 = math.radians(a[0]), math.radians(b[0]); dp = p2 - p1; dl = math.radians(b[1] - a[1])
    return 2 * R * math.asin(math.sqrt(math.sin(dp / 2) ** 2 + math.cos(p1) * math.cos(p2) * math.sin(dl / 2) ** 2))

def build(rid, stops, *, texts, biomes, story, facts, dangers=(), achievements, length_m=None, loop=False, underground=None):
    """stops: (id, kind, elev, lat, lon, en_name, en_text, uk_name, uk_text). Distances along the line, scaled to length_m."""
    pts = [(s[3], s[4]) for s in stops]
    cum = [0.0]
    for a, b in zip(pts, pts[1:]): cum.append(cum[-1] + hav(a, b))
    total = cum[-1]; L = float(length_m or round(total, -3)); k = L / total
    at = {s[0]: round(c * k) for s, c in zip(stops, cum)}
    at[stops[-1][0]] = L
    wps = []
    for s in stops:
        w = {"id": s[0], "at_m": at[s[0]], "kind": s[1], "lat": s[3], "lon": s[4]}
        if s[2] is not None: w["elevation_m"] = s[2]
        wps.append(w)
    pos = lambda ref: at[ref] if isinstance(ref, str) else round(ref * L)   # a stop id or a fraction of the way
    route = {"id": rid, "version": 1, "default_locale": "en", "length_m": L}
    if loop: route["loop"] = True
    route["track"] = [[round(a, 5), round(b, 5)] for a, b in pts]
    route["waypoints"] = wps
    route["biomes"] = [{"at_m": pos(r), "type": t} for r, t in biomes]
    route["story"] = [{"id": i, "at_m": pos(r)} for i, r, _, _ in story]
    route["facts"] = [{"id": i, "at_m": pos(r)} for i, r, _, _ in facts]
    if dangers: route["dangers"] = [{"id": i, "at_m": pos(r)} for i, r, _, _ in dangers]
    if underground: route["underground"] = [{"from_m": pos(a), "to_m": pos(b)} for a, b in underground]
    route["achievements"] = [{"id": i, "rule": rule} for i, rule, _, _ in achievements]
    d = os.path.join(OUT, rid, "pack", rid)
    shutil.rmtree(os.path.join(OUT, rid, "pack"), ignore_errors=True)
    os.makedirs(os.path.join(d, "locales"))
    json.dump(route, open(os.path.join(d, "route.json"), "w"), ensure_ascii=False, indent=2)
    for lang, li in (("en", 0), ("uk", 1)):
        loc = {"name": texts[lang][0], "description": texts[lang][1],
               "waypoints": {s[0]: {"name": s[5 + 2 * li], "text": s[6 + 2 * li]} for s in stops},
               "story": {i: (en, uk)[li] for i, _, en, uk in story},
               "facts": {i: (en, uk)[li] for i, _, en, uk in facts},
               "achievements": {i: {"name": (en, uk)[li][0], "description": (en, uk)[li][1]} for i, _, en, uk in achievements}}
        if dangers: loc["dangers"] = {i: (en, uk)[li] for i, _, en, uk in dangers}
        json.dump(loc, open(os.path.join(d, "locales", f"{lang}.json"), "w"), ensure_ascii=False, indent=2)
    zpath = os.path.join(OUT, rid, f"{rid}.zip")
    with zipfile.ZipFile(zpath, "w", zipfile.ZIP_DEFLATED) as z:
        for root, _, files in os.walk(d):
            for f in files:
                p = os.path.join(root, f); z.write(p, os.path.relpath(p, os.path.dirname(d)))
    with tempfile.TemporaryDirectory() as home:
        res = json.loads(subprocess.run([CORE, "route", "import", "--path", zpath], env={"COMMIT_HIKE_HOME": home},
                                        capture_output=True, text=True).stdout)
    if not res.get("ok"): raise SystemExit(f"{rid}: {res['error']['message']}")
    print(f"{rid}: valid, {L/1000:.0f} km, {len(wps)} stops, {len(route['facts'])} facts, {len(route.get('dangers', []))} dangers")
    return route

A = lambda en, uk: (en, uk)

# ================================================================ West Highland Way
build("west-highland-way", [
 ("milngavie", "start", 50, 55.9422, -4.3136, "Milngavie", "The obelisk in the town centre marks the start. 154 km of Scotland ahead.", "Мілнгаві", "Обеліск у центрі містечка — старт. Попереду 154 км Шотландії."),
 ("drymen", "village", 60, 56.0656, -4.4533, "Drymen", "A village pub and the first view of the Highland hills.", "Драймен", "Сільський паб і перший погляд на гори Гайленду."),
 ("conic-hill", "pass", 361, 56.0960, -4.5120, "Conic Hill", "From the top, Loch Lomond and its islands lie along the Highland Boundary Fault.", "Конік-Гілл", "З вершини видно Лох-Ломонд і його острови вздовж Гайлендського розлому."),
 ("balmaha", "village", 20, 56.0846, -4.5402, "Balmaha", "Boats, a lochside inn and the West Highland Way's busiest beach.", "Балмаха", "Човни, корчма над озером і найлюдніший пляж маршруту."),
 ("rowardennan", "hut", 30, 56.1528, -4.6403, "Rowardennan", "Ben Lomond rises behind the youth hostel.", "Роварденнан", "За молодіжним хостелом здіймається Бен-Ломонд."),
 ("inversnaid", "landmark", 30, 56.2445, -4.6893, "Inversnaid", "A waterfall crashes into the loch next to the hotel.", "Інверснейд", "Біля готелю в озеро обрушується водоспад."),
 ("inverarnan", "village", 20, 56.3315, -4.7213, "Inverarnan", "The Drovers Inn has fed walkers and cattle drovers since 1705.", "Інверарнан", "Корчма гуртівників годує мандрівників і погоничів худоби з 1705 року."),
 ("tyndrum", "village", 250, 56.4356, -4.7130, "Tyndrum", "Halfway, and the start of the big mountains.", "Тіндрум", "Половина шляху й початок великих гір."),
 ("bridge-of-orchy", "bridge", 190, 56.5160, -4.7700, "Bridge of Orchy", "A stone bridge from the 1750s over the River Orchy.", "Брідж-оф-Оркі", "Кам'яний міст 1750-х над річкою Оркі."),
 ("rannoch-moor", "landmark", 330, 56.6230, -4.8100, "Rannoch Moor", "Bog, lochans and sky: one of the last great wildernesses in Britain.", "Болото Ранох", "Трясовина, маленькі озерця й небо — одна з останніх великих диких місцин Британії."),
 ("kingshouse", "hut", 250, 56.6450, -4.8460, "Kingshouse", "Red deer graze outside one of Scotland's oldest inns, under Buachaille Etive Mòr.", "Кінгсгаус", "Олені пасуться біля одного з найстаріших заїздів Шотландії, під горою Бухалле-Етів-Мор."),
 ("devils-staircase", "pass", 548, 56.6760, -4.9300, "The Devil's Staircase", "The highest point of the Way, zigzagging out of Glen Coe.", "Чортові сходи", "Найвища точка маршруту — серпантин угору з долини Гленко."),
 ("kinlochleven", "village", 10, 56.7140, -4.9640, "Kinlochleven", "Down at sea level by a long sea loch, once a town of aluminium.", "Кінлохлевен", "Унизу, біля довгої морської затоки; колись тут виплавляли алюміній."),
 ("lairigmor", "pass", 330, 56.7350, -5.0400, "Lairigmor", "An old military road through an empty glen.", "Ларіґмор", "Стара військова дорога порожньою долиною."),
 ("fort-william", "finish", 20, 56.8198, -5.1052, "Fort William", "The end of the Way, under Ben Nevis, the highest mountain in Britain.", "Форт-Вільям", "Кінець шляху, під Бен-Невісом — найвищою горою Британії."),
], texts={"en": ("West Highland Way", "Scotland's best-loved long-distance trail: 154 km from the edge of Glasgow along Loch Lomond, across Rannoch Moor and through Glen Coe to Fort William, under Ben Nevis. The track is simplified: a line through the stops."),
          "uk": ("Західно-Гайлендський шлях", "Найулюбленіша довга стежка Шотландії: 154 км від околиць Глазго вздовж Лох-Ломонду, через болото Ранох і долину Гленко до Форт-Вільяма під Бен-Невісом. Трек спрощений — лінія через зупинки.")},
   length_m=154000,
   biomes=[("milngavie", "grove"), ("drymen", "fields"), ("conic-hill", "meadow"), ("balmaha", "coast"), ("inverarnan", "forest"),
           ("tyndrum", "tundra"), ("rannoch-moor", "swamp"), ("kingshouse", "rock"), ("kinlochleven", "forest"), ("lairigmor", "tundra"), ("fort-william", "village")],
   story=[("rain", 0.3, "Rain comes sideways off the loch, then stops as if it never started.", "Дощ летить з озера навскіс, а потім зникає, ніби його й не було."),
          ("midges", 0.5, "The midges find you the moment the wind drops.", "Мошка знаходить тебе, щойно стихає вітер."),
          ("glencoe", 0.75, "The walls of Glen Coe close in, dark and enormous.", "Стіни Гленко сходяться — темні й величезні.")],
   facts=[("opened", 0.02, "The West Highland Way opened in 1980 as Scotland's first official long-distance route.", "Західно-Гайлендський шлях відкрили 1980 року як перший офіційний довгий маршрут Шотландії."),
          ("fault", "conic-hill", "Conic Hill sits on the Highland Boundary Fault: from its top you see the Lowlands on one side and the Highlands on the other.", "Конік-Гілл стоїть на Гайлендському розломі: з вершини з одного боку видно Низовину, з другого — Високогір'я."),
          ("loch-lomond", "rowardennan", "Loch Lomond is the largest lake in Great Britain by surface area.", "Лох-Ломонд — найбільше озеро Великої Британії за площею."),
          ("drovers", "inverarnan", "Drovers walked cattle along these glens to the markets of the south for centuries.", "Століттями гуртівники гнали цими долинами худобу на ринки півдня."),
          ("ben-nevis", "fort-william", "Ben Nevis, 1,345 m, is the highest mountain in the British Isles.", "Бен-Невіс, 1345 м, — найвища гора Британських островів.")],
   achievements=[("whw-start", {"type": "distance", "min_m": 1000}, A("Off from Milngavie", "Walk your first kilometre."), A("Вирушили з Мілнгаві", "Пройди перший кілометр.")),
                 ("whw-loch", {"type": "waypoint", "waypoint": "rowardennan"}, A("Bonnie Banks", "Walk along Loch Lomond to Rowardennan."), A("Береги Лох-Ломонду", "Пройди вздовж Лох-Ломонду до Роварденнана.")),
                 ("whw-moor", {"type": "waypoint", "waypoint": "rannoch-moor"}, A("Across the Moor", "Reach Rannoch Moor."), A("Через болото", "Дійди до болота Ранох.")),
                 ("whw-staircase", {"type": "waypoint", "waypoint": "devils-staircase"}, A("The Devil's Staircase", "Climb out of Glen Coe."), A("Чортові сходи", "Піднімися з Гленко.")),
                 ("whw-finish", {"type": "finish"}, A("Fort William", "Walk the whole Way."), A("Форт-Вільям", "Пройди весь шлях.")),
                 ("whw-streak", {"type": "streak", "days": 7}, A("A Week in the Highlands", "Commit seven days in a row."), A("Тиждень у Гайленді", "Коміть сім днів поспіль."))])

# ================================================================ The Odyssey
build("the-odyssey", [
 ("troy", "start", 30, 39.9575, 26.2389, "Troy", "Ten years of war are over. Odysseus sets sail for home.", "Троя", "Десять років війни позаду. Одіссей вирушає додому."),
 ("ismarus", "village", 10, 40.8700, 25.5300, "Ismarus", "The city of the Cicones, raided too greedily: the first of many losses.", "Ісмар", "Місто кіконів, пограбоване надто жадібно, — перша з багатьох втрат."),
 ("malea", "landmark", 0, 36.4400, 23.2000, "Cape Malea", "A north wind catches the ships at the southern tip of Greece and blows them off every map.", "Мис Малея", "Північний вітер підхоплює кораблі біля південного краю Греції й зносить їх за межі всіх мап."),
 ("lotus", "landmark", 0, 33.8100, 10.8600, "Land of the Lotus-eaters", "Whoever eats the lotus forgets the way home. Some crew have to be dragged back to the ships.", "Країна лотофагів", "Хто скуштує лотос, забуває дорогу додому. Декого з команди доводиться тягти на кораблі силоміць."),
 ("cyclops", "cave", 100, 37.5600, 15.1700, "The Cyclops' cave", "Odysseus tells the one-eyed giant his name is Nobody.", "Печера циклопа", "Одіссей каже одноокому велетню, що його звуть Ніхто."),
 ("aeolia", "lighthouse", 0, 38.4700, 14.9500, "Aeolia", "The keeper of the winds gives Odysseus a bag with every wind but the one that blows home.", "Еолія", "Володар вітрів дає Одіссеєві мішок з усіма вітрами, крім того, що дме додому."),
 ("laestrygonians", "harbor", 0, 41.3900, 9.1600, "The harbour of the Laestrygonians", "A narrow harbour between cliffs, and giants throwing rocks from above.", "Гавань лестригонів", "Вузька гавань між скелями — і велетні, що жбурляють каміння згори."),
 ("aeaea", "landmark", 500, 41.2300, 13.0600, "Aeaea, Circe's island", "The sorceress turns the crew into pigs, then helps them on their way.", "Ея, острів Кірки", "Чарівниця перетворює команду на свиней, а потім допомагає рушити далі."),
 ("underworld", "cave", 0, 40.8400, 14.0800, "The edge of the Underworld", "At a dark lake Odysseus speaks with the shades of the dead.", "Край Аїду", "Біля темного озера Одіссей розмовляє з тінями померлих."),
 ("sirens", "landmark", 0, 40.5800, 14.4300, "The Sirens", "Wax in the crew's ears, Odysseus tied to the mast, listening.", "Сирени", "Віск у вухах команди, Одіссей прив'язаний до щогли — і слухає."),
 ("scylla", "pass", 0, 38.2500, 15.6300, "Scylla and Charybdis", "A monster on one side of the strait, a whirlpool on the other.", "Скілла й Харибда", "З одного боку протоки — чудовисько, з другого — вир."),
 ("thrinacia", "landmark", 50, 36.7000, 15.1000, "Thrinacia", "The cattle of the Sun graze here. They must not be touched.", "Тринакія", "Тут пасуться корови Геліоса. Їх не можна чіпати."),
 ("ogygia", "landmark", 50, 36.0500, 14.2500, "Ogygia, Calypso's island", "Seven years on the nymph's island, until the gods order her to let him go.", "Огігія, острів Каліпсо", "Сім років на острові німфи, поки боги не наказують відпустити його."),
 ("scheria", "harbor", 20, 39.6200, 19.9200, "Scheria", "Shipwrecked and alone, Odysseus tells his story at the Phaeacian court.", "Схерія", "Корабельна аварія, він сам; при дворі феаків Одіссей розповідає свою історію."),
 ("ithaca", "finish", 100, 38.3700, 20.7200, "Ithaca", "Home at last, twenty years after he left. Only his old dog knows him at once.", "Ітака", "Нарешті вдома — через двадцять років. Одразу впізнає його лише старий пес."),
], texts={"en": ("The Odyssey", "Homer's voyage home from Troy to Ithaca, on a real map of the Mediterranean. Where Homer's places really were has been argued about since antiquity; the stops follow traditional identifications. A story route: 3,000 years old and free for everyone."),
          "uk": ("Одіссея", "Гомерова мандрівка додому — від Трої до Ітаки на справжній мапі Середземномор'я. Де насправді були Гомерові місця, сперечаються ще з античності; зупинки йдуть за традиційними ототожненнями. Маршрут-історія, якій 3000 років і яка вільна для всіх.")},
   biomes=[("troy", "fields"), ("ismarus", "coast"), ("malea", "water"), ("lotus", "desert"), ("cyclops", "volcanic"), ("aeolia", "water"),
           ("laestrygonians", "rock"), ("aeaea", "grove"), ("underworld", "swamp"), ("sirens", "coast"), ("scylla", "water"), ("thrinacia", "meadow"),
           ("ogygia", "coast"), ("scheria", "grove"), ("ithaca", "village")],
   story=[("wine-dark", 0.12, "The sea turns the colour Homer called wine-dark as the sun goes down.", "Море стає таким, яке Гомер називав винно-темним, коли сідає сонце."),
          ("crew", 0.6, "Fewer oars on every crossing. You stop counting.", "З кожною переправою весел менше. Ти перестаєш рахувати."),
          ("olive", 0.97, "The smell of olive trees from the shore you left twenty years ago.", "Запах олив із берега, який ти покинув двадцять років тому.")],
   facts=[("hisarlik", "troy", "Heinrich Schliemann dug at Hisarlik in Turkey from 1870; the site is now accepted as Troy and is on the World Heritage List.", "Генріх Шліман копав на пагорбі Гіссарлик у Туреччині з 1870 року; тепер це місце визнають Троєю, воно в списку Світової спадщини."),
          ("djerba", "lotus", "Ancient writers already placed the Lotus-eaters on Djerba, an island off Tunisia.", "Ще античні автори поміщали лотофагів на Джербу — острів біля Тунісу."),
          ("aeolian", "aeolia", "The Aeolian Islands north of Sicily are named after Aeolus, keeper of the winds.", "Еолові острови на північ від Сицилії названі на честь Еола, володаря вітрів."),
          ("circeo", "aeaea", "Monte Circeo on the Italian coast keeps Circe's name; seen from the sea it looks like an island.", "Гора Чірчео на італійському узбережжі носить ім'я Кірки; з моря вона схожа на острів."),
          ("messina", "scylla", "The Strait of Messina really has strong currents and whirlpools where two seas meet.", "У Мессінській протоці справді сильні течії й вири там, де зустрічаються два моря."),
          ("nobody", "cyclops", "The Odyssey has about 12,000 lines in 24 books and was composed around the 8th century BC.", "«Одіссея» має близько 12 000 рядків у 24 піснях і склалася приблизно у VIII столітті до н. е.")],
   dangers=[("polyphemus", "cyclops", "The giant blocks the cave with a boulder. Hide under the sheep.", "Велетень закриває печеру брилою. Ховайся під вівцями."),
            ("whirlpool", "scylla", "Charybdis swallows the sea. Steer close to the cliff and lose six men.", "Харибда ковтає море. Тримайся скелі — і втрать шістьох."),
            ("storm", 0.84, "Zeus's thunderbolt breaks the last ship. Cling to the mast.", "Блискавка Зевса розбиває останній корабель. Тримайся за щоглу.")],
   achievements=[("ody-sail", {"type": "distance", "min_m": 10000}, A("Oars Out", "Sail your first 10 km."), A("Весла на воду", "Пропливи перші 10 км.")),
                 ("ody-nobody", {"type": "waypoint", "waypoint": "cyclops"}, A("Nobody", "Escape the Cyclops."), A("Ніхто", "Втечи від циклопа.")),
                 ("ody-sirens", {"type": "waypoint", "waypoint": "sirens"}, A("Tied to the Mast", "Sail past the Sirens."), A("Прив'язаний до щогли", "Пропливи повз сирен.")),
                 ("ody-strait", {"type": "waypoint", "waypoint": "scylla"}, A("Between Two Evils", "Get through between Scylla and Charybdis."), A("Між двох лих", "Пройди між Скіллою й Харибдою.")),
                 ("ody-home", {"type": "finish"}, A("Ithaca", "Come home."), A("Ітака", "Повернися додому.")),
                 ("ody-patience", {"type": "streak", "days": 20}, A("Twenty Years", "Commit 20 days in a row."), A("Двадцять років", "Коміть 20 днів поспіль."))])

# ================================================================ Journey to the Centre of the Earth
crater = (64.8080, -23.7760); strom = (38.7939, 15.2133)
lerp = lambda f: (round(crater[0] + (strom[0] - crater[0]) * f, 4), round(crater[1] + (strom[1] - crater[1]) * f, 4))
r = build("journey-to-the-centre-of-the-earth", [
 ("hamburg", "start", 10, 53.5511, 9.9937, "Hamburg", "Professor Lidenbrock deciphers a runic note: a descent to the centre of the Earth begins in Iceland.", "Гамбург", "Професор Лінденброк розшифровує рунічну записку: спуск до центру Землі починається в Ісландії."),
 ("copenhagen", "harbor", 5, 55.6761, 12.5683, "Copenhagen", "The professor makes his nephew Axel climb a church spire to cure his fear of heights.", "Копенгаген", "Професор змушує небожа Акселя лізти на шпиль церкви, щоб вилікувати страх висоти."),
 ("reykjavik", "harbor", 20, 64.1466, -21.9426, "Reykjavík", "They hire Hans, a calm Icelandic guide who never hurries.", "Рейк'явік", "Вони наймають Ганса — спокійного ісландського провідника, який ніколи не поспішає."),
 ("stapi", "village", 30, 64.7810, -23.6270, "Stapi", "A village under the glacier, the last houses before the volcano.", "Стапі", "Село під льодовиком — останні хати перед вулканом."),
 ("crater", "volcano", 1446, crater[0], crater[1], "Snæfellsjökull", "At the end of June the shadow of a peak points to the right chimney. Down they go.", "Снайфедльсйокюдль", "Наприкінці червня тінь вершини вказує на потрібний отвір. Вони спускаються."),
 ("hans-stream", "river", 400, *lerp(0.12), "Hans's Stream", "Dying of thirst, they hear water behind the rock. Hans opens a spring.", "Струмок Ганса", "Помираючи від спраги, вони чують воду за скелею. Ганс відкриває джерело."),
 ("mushroom-forest", "landmark", 200, *lerp(0.3), "The forest of giant mushrooms", "Mushrooms as tall as trees, under a sky of glowing gas.", "Ліс велетенських грибів", "Гриби заввишки з дерева під небом із сяйного газу."),
 ("lidenbrock-sea", "lake", 0, *lerp(0.4), "The Lidenbrock Sea", "An ocean inside the Earth. They build a raft.", "Море Лінденброка", "Океан усередині Землі. Вони будують пліт."),
 ("axel-island", "landmark", 0, *lerp(0.55), "Axel Island", "A geyser island in the middle of the underground sea.", "Острів Акселя", "Острів із гейзером посеред підземного моря."),
 ("saknussemm", "cave", 300, *lerp(0.78), "Saknussemm's mark", "Two runic initials carved in the rock: someone was here three centuries ago.", "Знак Сакнуссема", "Два рунічні ініціали, вирізані на скелі: хтось був тут три століття тому."),
 ("stromboli", "volcano", 924, strom[0], strom[1], "Stromboli", "Thrown out of a volcano in Sicily's islands, singed but alive.", "Стромболі", "Викинуті з вулкана на островах біля Сицилії — обсмалені, але живі."),
 ("messina", "harbor", 10, 38.1938, 15.5540, "Messina", "Back among people, pretending to be shipwrecked sailors.", "Мессіна", "Знову серед людей — удають, що вони потерпілі моряки."),
 ("home", "finish", 10, 53.5600, 10.0100, "Home in Hamburg", "A compass that points south explains everything. Axel marries Gräuben.", "Знову в Гамбурзі", "Компас, що показує на південь, пояснює все. Аксель одружується з Ґраубен."),
], texts={"en": ("Journey to the Centre of the Earth", "Jules Verne's novel of 1864 on a real map: from Hamburg to Iceland, down the crater of Snæfellsjökull, across an underground sea, and out of Stromboli in Italy. The underground stretch is walked in a tunnel. A story route, free for everyone."),
          "uk": ("Подорож до центру Землі", "Роман Жуля Верна 1864 року на справжній мапі: з Гамбурга до Ісландії, вниз кратером Снайфедльсйокюдля, через підземне море і назовні з вулкана Стромболі в Італії. Підземну частину проходиш тунелем. Маршрут-історія, вільна для всіх.")},
   loop=True,
   biomes=[("hamburg", "village"), ("copenhagen", "coast"), ("reykjavik", "tundra"), ("stapi", "rock"), ("crater", "snow"), ("hans-stream", "rock"),
           ("mushroom-forest", "swamp"), ("lidenbrock-sea", "water"), ("saknussemm", "rock"), ("stromboli", "volcanic"), ("messina", "coast"), ("home", "village")],
   underground=[("crater", "stromboli")],
   story=[("dark", 0.22, "The lamps are the only light for weeks. Axel counts the days by the professor's watch.", "Тижнями єдине світло — лампи. Аксель рахує дні за годинником професора."),
          ("raft", 0.42, "The raft creaks under a sky with no sun.", "Пліт рипить під небом без сонця."),
          ("heat", 0.8, "The walls get hotter. The compass spins.", "Стіни стають гарячішими. Компас крутиться.")],
   facts=[("verne", "hamburg", "Jules Verne published the novel in 1864; it has been in the public domain for over a century.", "Жуль Верн видав роман 1864 року; понад сто років він у суспільному надбанні."),
          ("snaefells", "crater", "Snæfellsjökull is a real glacier-capped volcano, 1,446 m, on Iceland's west coast; on clear days it is seen from Reykjavík.", "Снайфедльсйокюдль — справжній вулкан під льодовиком, 1446 м, на західному узбережжі Ісландії; у ясну днину його видно з Рейк'явіка."),
          ("geysir", "axel-island", "The word geyser comes from Geysir, a hot spring in Iceland.", "Слово «гейзер» походить від Ґейсіра — гарячого джерела в Ісландії."),
          ("lighthouse", "stromboli", "Stromboli has erupted almost constantly for thousands of years; sailors call it the Lighthouse of the Mediterranean.", "Стромболі вивергається майже безперервно тисячі років; моряки звуть його Маяком Середземномор'я."),
          ("depth", 0.5, "The deepest hole humans ever drilled, the Kola Superdeep Borehole, reached about 12 km: far from the centre, 6,371 km down.", "Найглибша свердловина, яку пробурили люди, — Кольська надглибока, близько 12 км. До центру Землі — 6371 км.")],
   dangers=[("thirst", "hans-stream", "No water for days. The professor shares his last drops.", "Кілька днів без води. Професор ділиться останніми краплями."),
            ("monsters", 0.48, "Two sea monsters fight around the raft.", "Два морські чудовиська б'ються довкола плоту."),
            ("eruption", 0.97, "The raft rides a column of lava up the chimney.", "Пліт мчить угору шахтою на стовпі лави.")],
   achievements=[("jce-start", {"type": "distance", "min_m": 5000}, A("Runes Decoded", "Travel your first 5 km."), A("Руни розшифровано", "Подолай перші 5 км.")),
                 ("jce-descent", {"type": "waypoint", "waypoint": "crater"}, A("Into the Crater", "Reach Snæfellsjökull."), A("У кратер", "Дійди до Снайфедльсйокюдля.")),
                 ("jce-sea", {"type": "waypoint", "waypoint": "lidenbrock-sea"}, A("An Ocean Below", "Reach the Lidenbrock Sea."), A("Океан унизу", "Дійди до моря Лінденброка.")),
                 ("jce-out", {"type": "waypoint", "waypoint": "stromboli"}, A("Out of the Volcano", "Come out of Stromboli."), A("З вулкана", "Вийди зі Стромболі.")),
                 ("jce-home", {"type": "finish"}, A("Home Again", "Get back to Hamburg."), A("Знову вдома", "Повернися до Гамбурга.")),
                 ("jce-commits", {"type": "commits", "count": 300}, A("Professor's Patience", "Make 300 commits on the way."), A("Терпіння професора", "Зроби 300 комітів у дорозі."))])
