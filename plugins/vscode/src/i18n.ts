// Plugin messages (notifications, pickers). The language follows the core's
// setting (Commit Hike menu → Language), else the editor's display language.
// Command titles in the Command Palette come from package.nls*.json and follow
// VS Code's own display language.

const en = {
  cantStart: "Commit Hike can't start: {0}",
  rateAsk: "Enjoying Commit Hike? A rating on {0} helps other developers find it. It takes a minute.",
  rateYes: "Rate it", rateLater: "Later", rateNever: "Don't ask again",
  showTrail: "Show trail", chooseTrail: "Choose trail",
  finished: "You finished {0}! Pick your next trail.",
  reached: "You reached {0}.",
  rewritten: "History was rewritten here, so this project was recounted from git.",
  barSetup: "Set up Commit Hike to turn your commits into a journey.",
  barOf: "{0} of {1}", barCompleted: "Trail completed.", barNext: "Next stop: {0}, in {1}", barToday: "Today: {0}",
  barAltitude: "Altitude: {0} m, climbed {1} m", barProject: "This project: {0}, {1}", barNotCounted: "This project isn't counted.",
  offer: "Commit Hike turns your commits into a hiking journey. It never writes to your projects and keeps everything on this computer.",
  setUp: "Set up", notNow: "Not now",
  setupTitle: "Set up Commit Hike",
  modeAll: "Count all my projects", modeAllDetail: "Commits in any repository you open move you forward.",
  modeSelected: "Only projects I choose", modeSelectedDetail: "You turn counting on for each project.",
  modeQuestion: "Which projects should count?",
  histYes: "Include commits I already made", histYesDetail: "Your history counts, so you may start partway along the trail.",
  histNo: "Start from today", histNoDetail: "Only new commits count.", histQuestion: "Where does your journey start?",
  emailsPrompt: "Commits with these author emails count as yours. Separate several with commas.",
  emailsInvalid: "Enter at least one email address.",
  countThis: "Count commits in this project?", countIt: "Count it",
  counting: "Commit Hike: counting your commits",
  openRepoForTrail: "Open a file from a git repository to choose a trail for that project.",
  noProjectTrail: "No trail for this project", noProjectTrailDetail: "Commits here still count toward your main journey.",
  importItem: "$(cloud-download) Import a route or map…", templateItem: "$(new-file) Create a route template…",
  trailAll: "Trail for all projects", trailProject: "Trail for this project",
  histNow: "Start from now", journeyStart: "Where does this journey start?",
  openRepoFirst: "Open a file from a git repository first.",
  allCounted: "All your projects are already counted. To choose projects one by one, run “Commit Hike: Set Up”.",
  recounting: "Commit Hike: recounting this project from git history",
  recountSame: "Recount finished: everything already matched your git history.",
  recountDone: "Recount finished: {0} added, {1} corrected, {2} removed.",
  importTitle: "Import a route", importWhere: "Where is the route?", fromZip: "From a .zip file", fromFolder: "From a folder",
  importBtn: "Import", routePack: "Route pack", replaceQ: "{0}. Replace it?", replace: "Replace",
  imported: "Imported {0}: {1}, {2} stops.", walkNow: "Walk it now",
  whereCreate: "Where to create the route", createHere: "Create here", newRoute: "New route",
  routeIdPrompt: "Route id: lowercase letters, digits and dashes.", routeIdInvalid: "Use lowercase letters, digits and single dashes.",
  templateCreated: "Route template created. Edit route.json, locales/*.json and assets/, then import the folder.", importNow: "Import now",
  noCustom: "You haven't imported any routes.", removeTitle: "Remove a route",
  removeHint: "Progress you made stays; you just can't choose the route anymore.", removed: "{0} was removed.",
  iconTitle: "Choose a hiker icon (PNG with a transparent background, up to 512×512, facing right)",
  useAsHiker: "Use as hiker", pngImage: "PNG image",
  langTitle: "Commit Hike language", langAuto: "Same as the editor", langCurrent: "current",
  teamOn: "Teammates are now shown on this project's trail. Names are read from git history and never stored.",
  teamOff: "Teammates are hidden.", teamNeedsRepo: "Open a file from a git repository to see its teammates.",
  error: "Commit Hike: {0}",
  diffTitle: "Commit Hike difficulty", diffQuestion: "How fast should commits move you?",
  diffHint: "Applies to commits from now on; distance already walked stays.",
  diff_easy: "Easy", diff_medium: "Medium", diff_hard: "Hard", diffDetail: "A typical day of commits: about {0}",
};
type Key = keyof typeof en;

const uk: Record<Key, string> = {
  rateAsk: "Подобається Commit Hike? Оцінка на {0} допоможе іншим розробникам його знайти. Це займе хвилину.",
  rateYes: "Оцінити", rateLater: "Пізніше", rateNever: "Більше не питати",
  cantStart: "Commit Hike не може запуститися: {0}",
  showTrail: "Показати стежку", chooseTrail: "Вибрати стежку",
  finished: "Ти пройшов маршрут «{0}»! Вибери наступну стежку.",
  reached: "Ти дійшов до: {0}.",
  rewritten: "Тут переписали історію, тому проєкт перераховано з git.",
  barSetup: "Налаштуй Commit Hike, щоб перетворити коміти на подорож.",
  barOf: "{0} з {1}", barCompleted: "Стежку пройдено.", barNext: "Наступна зупинка: {0}, через {1}", barToday: "Сьогодні: {0}",
  barAltitude: "Висота: {0} м, набрано {1} м", barProject: "Цей проєкт: {0}, {1}", barNotCounted: "Цей проєкт не враховується.",
  offer: "Commit Hike перетворює коміти на похід. Він нічого не записує у твої проєкти й тримає все на цьому комп'ютері.",
  setUp: "Налаштувати", notNow: "Не зараз",
  setupTitle: "Налаштування Commit Hike",
  modeAll: "Рахувати всі мої проєкти", modeAllDetail: "Коміти в будь-якому відкритому репозиторії просувають тебе.",
  modeSelected: "Лише вибрані проєкти", modeSelectedDetail: "Ти вмикаєш підрахунок для кожного проєкту.",
  modeQuestion: "Які проєкти рахувати?",
  histYes: "Врахувати вже зроблені коміти", histYesDetail: "Історія рахується, тож можеш почати не з початку стежки.",
  histNo: "Почати з сьогодні", histNoDetail: "Рахуються лише нові коміти.", histQuestion: "Звідки почнеться подорож?",
  emailsPrompt: "Коміти з цими email авторів рахуються як твої. Кілька адрес — через кому.",
  emailsInvalid: "Введи хоча б одну email-адресу.",
  countThis: "Рахувати коміти в цьому проєкті?", countIt: "Рахувати",
  counting: "Commit Hike: рахуємо твої коміти",
  openRepoForTrail: "Відкрий файл із git-репозиторію, щоб вибрати стежку для цього проєкту.",
  noProjectTrail: "Без стежки для цього проєкту", noProjectTrailDetail: "Коміти тут і далі рахуються в основну подорож.",
  importItem: "$(cloud-download) Імпортувати маршрут або карту…", templateItem: "$(new-file) Створити шаблон маршруту…",
  trailAll: "Стежка для всіх проєктів", trailProject: "Стежка для цього проєкту",
  histNow: "Почати зараз", journeyStart: "Звідки почнеться ця подорож?",
  openRepoFirst: "Спершу відкрий файл із git-репозиторію.",
  allCounted: "Усі твої проєкти вже рахуються. Щоб вибирати проєкти по одному, запусти «Commit Hike: Set Up».",
  recounting: "Commit Hike: перераховуємо проєкт з історії git",
  recountSame: "Перерахунок завершено: усе вже збігалося з історією git.",
  recountDone: "Перерахунок завершено: додано {0}, виправлено {1}, видалено {2}.",
  importTitle: "Імпорт маршруту", importWhere: "Де лежить маршрут?", fromZip: "У .zip-файлі", fromFolder: "У папці",
  importBtn: "Імпортувати", routePack: "Пакет маршруту", replaceQ: "{0}. Замінити?", replace: "Замінити",
  imported: "Імпортовано «{0}»: {1}, зупинок: {2}.", walkNow: "Іти зараз",
  whereCreate: "Де створити маршрут", createHere: "Створити тут", newRoute: "Новий маршрут",
  routeIdPrompt: "Id маршруту: малі літери, цифри й дефіси.", routeIdInvalid: "Використовуй малі літери, цифри й одинарні дефіси.",
  templateCreated: "Шаблон маршруту створено. Відредагуй route.json, locales/*.json та assets/, потім імпортуй папку.", importNow: "Імпортувати",
  noCustom: "Ти ще не імпортував жодного маршруту.", removeTitle: "Видалити маршрут",
  removeHint: "Твій прогрес лишиться; просто маршрут більше не можна буде вибрати.", removed: "«{0}» видалено.",
  iconTitle: "Вибери іконку мандрівника (PNG з прозорим фоном, до 512×512, обличчям праворуч)",
  useAsHiker: "Зробити мандрівником", pngImage: "Зображення PNG",
  langTitle: "Мова Commit Hike", langAuto: "Як у редакторі", langCurrent: "зараз",
  teamOn: "Тепер на стежці цього проєкту видно команду. Імена беруться з історії git і ніде не зберігаються.",
  teamOff: "Команду сховано.", teamNeedsRepo: "Відкрий файл із git-репозиторію, щоб побачити його команду.",
  error: "Commit Hike: {0}",
  diffTitle: "Складність Commit Hike", diffQuestion: "Як швидко коміти мають тебе просувати?",
  diffHint: "Діє для комітів відтепер; уже пройдене лишається.",
  diff_easy: "Легка", diff_medium: "Середня", diff_hard: "Складна", diffDetail: "Типовий день комітів: близько {0}",
};

const catalogs: Record<string, Record<Key, string>> = { en, uk };
let current: Record<Key, string> = en;
let currentLang = "en";

export function setLanguage(tag: string | undefined): void {
  const base = (tag || "en").toLowerCase().split(/[-_]/)[0];
  currentLang = catalogs[base] ? base : "en";
  current = catalogs[currentLang];
}
export function language(): string { return currentLang; }

export function t(key: Key, ...args: (string | number)[]): string {
  return (current[key] ?? en[key]).replace(/\{(\d)\}/g, (_, i: string) => String(args[Number(i)] ?? ""));
}

export function formatDistanceL(m: number): string {
  const unit = currentLang === "uk" ? { m: "м", km: "км", dec: "," } : { m: "m", km: "km", dec: "." };
  if (m < 1000) return `${Math.round(m)} ${unit.m}`;
  const km = m / 1000;
  return `${km < 100 ? km.toFixed(1).replace(".", unit.dec) : Math.round(km)} ${unit.km}`;
}

export const LANGUAGE_NAMES: Record<string, string> = { en: "English", uk: "Українська" };
