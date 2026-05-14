/**
 * Curated challenge dataset for KNIRVANA game.
 * Each challenge maps to an ErrorNode type and gives LLM agents
 * a real debugging problem to solve.
 */

export type ErrorNodeType =
  | 'Memory Leak'
  | 'Logic Error'
  | 'Race Condition'
  | 'Buffer Overflow'
  | 'API Timeout';

export interface Challenge {
  id: string;
  title: string;
  type: ErrorNodeType;
  difficulty: number; // 0.0–1.0
  bounty: number;     // NRN reward
  description: string;
  buggyCode: string;
  context: string;    // Sentence describing expected behavior
  hints: string[];
}

export const CHALLENGES: Challenge[] = [
  // ── Memory Leak ──────────────────────────────────────────────────────────
  {
    id: 'ml-001',
    type: 'Memory Leak',
    title: 'Event Listener Accumulation',
    difficulty: 0.3,
    bounty: 30,
    description:
      'A React component registers a keydown listener inside useEffect on every render but never cleans it up, causing the listener count to grow unboundedly.',
    buggyCode: `function DataTable({ onRowClick, selectedRow }) {
  useEffect(() => {
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') onRowClick(selectedRow);
    });
  }, [selectedRow, onRowClick]);

  return <table>{/* rows */}</table>;
}`,
    context: 'The component should respond to Enter key presses without accumulating listeners.',
    hints: ['useEffect must return a cleanup function', 'Store the handler reference to remove it'],
  },
  {
    id: 'ml-002',
    type: 'Memory Leak',
    title: 'Interval Never Cleared',
    difficulty: 0.2,
    bounty: 20,
    description:
      'A polling function starts a setInterval when the component mounts but never clears it when the component unmounts.',
    buggyCode: `function LiveFeed({ url }) {
  const [data, setData] = useState(null);

  useEffect(() => {
    setInterval(async () => {
      const res = await fetch(url);
      setData(await res.json());
    }, 2000);
  }, [url]);

  return <div>{JSON.stringify(data)}</div>;
}`,
    context: 'Live feed should poll every 2 seconds and stop completely when unmounted.',
    hints: ['setInterval returns an ID', 'clearInterval must be called in the cleanup function'],
  },
  {
    id: 'ml-003',
    type: 'Memory Leak',
    title: 'Unresolved Promise After Unmount',
    difficulty: 0.5,
    bounty: 45,
    description:
      'An async fetch inside useEffect calls setState after the component has unmounted, causing a memory leak and React warning.',
    buggyCode: `function UserProfile({ userId }) {
  const [user, setUser] = useState(null);

  useEffect(() => {
    fetchUser(userId).then(data => setUser(data));
  }, [userId]);

  return <div>{user?.name}</div>;
}`,
    context: 'Profile should fetch user data and handle component unmount safely.',
    hints: ['Use an isMounted flag or AbortController', 'Check flag before calling setState'],
  },
  {
    id: 'ml-004',
    type: 'Memory Leak',
    title: 'Growing Cache Without Eviction',
    difficulty: 0.6,
    bounty: 55,
    description:
      'A memoization cache stores results in a plain object that grows without bound as unique keys accumulate.',
    buggyCode: `const cache: Record<string, unknown> = {};

function memoize<T>(fn: (key: string) => T): (key: string) => T {
  return (key: string) => {
    if (!(key in cache)) {
      cache[key] = fn(key);
    }
    return cache[key] as T;
  };
}`,
    context: 'Cache should limit stored entries to prevent unbounded memory growth.',
    hints: ['Use an LRU strategy or cap size', 'Map preserves insertion order for easy eviction'],
  },
  {
    id: 'ml-005',
    type: 'Memory Leak',
    title: 'WebSocket Not Closed on Unmount',
    difficulty: 0.4,
    bounty: 40,
    description:
      'A component opens a WebSocket connection but never closes it when the component unmounts or the URL changes.',
    buggyCode: `function LiveChart({ wsUrl }) {
  const [points, setPoints] = useState([]);

  useEffect(() => {
    const ws = new WebSocket(wsUrl);
    ws.onmessage = (e) => setPoints(prev => [...prev, JSON.parse(e.data)]);
  }, [wsUrl]);

  return <canvas />;
}`,
    context: 'Chart should stream data via WebSocket and close the connection when no longer needed.',
    hints: ['Return a cleanup from useEffect', 'Call ws.close() in the cleanup'],
  },

  // ── Logic Error ──────────────────────────────────────────────────────────
  {
    id: 'le-001',
    type: 'Logic Error',
    title: 'Off-by-One in Pagination',
    difficulty: 0.2,
    bounty: 20,
    description:
      'A pagination function returns one fewer item than expected because it uses strict less-than instead of less-than-or-equal.',
    buggyCode: `function paginate<T>(items: T[], page: number, pageSize: number): T[] {
  const start = page * pageSize;
  const end = start + pageSize - 1; // bug here
  return items.slice(start, end);
}`,
    context: 'paginate([0..9], 0, 5) should return [0,1,2,3,4].',
    hints: ['Array.slice end index is exclusive', 'end should equal start + pageSize'],
  },
  {
    id: 'le-002',
    type: 'Logic Error',
    title: 'Wrong Operator Precedence in Discount',
    difficulty: 0.3,
    bounty: 25,
    description:
      'A discount calculation applies the percentage incorrectly due to missing parentheses around the subtraction.',
    buggyCode: `function applyDiscount(price: number, discountPercent: number): number {
  return price - price * discountPercent / 100;
  // Intended: price * (1 - discountPercent / 100)
  // But for price=100, discount=20 → returns 80, seems ok?
  // For price=100, discount=110 → returns -10, should clamp to 0
}`,
    context: 'Discounted price should never go below 0.',
    hints: ['Add Math.max(0, ...) to clamp', 'Consider edge cases above 100% discount'],
  },
  {
    id: 'le-003',
    type: 'Logic Error',
    title: 'Floating Point Comparison',
    difficulty: 0.4,
    bounty: 35,
    description:
      'A balance check uses strict equality to compare floating point numbers, causing valid transactions to be incorrectly rejected.',
    buggyCode: `function canAfford(balance: number, price: number): boolean {
  return balance === price || balance > price;
}

// canAfford(0.1 + 0.2, 0.3) returns false due to float imprecision`,
    context: 'canAfford should return true whenever balance is greater than or equal to price.',
    hints: ['Use Math.abs(a - b) < epsilon for float equality', 'Or use >= directly which already handles this'],
  },
  {
    id: 'le-004',
    type: 'Logic Error',
    title: 'Mutating State Directly in Reducer',
    difficulty: 0.35,
    bounty: 30,
    description:
      'A Redux-style reducer mutates the existing state array instead of returning a new one, causing missed re-renders.',
    buggyCode: `function cartReducer(state: CartItem[], action: CartAction): CartItem[] {
  switch (action.type) {
    case 'ADD_ITEM':
      state.push(action.item); // direct mutation
      return state;
    case 'REMOVE_ITEM':
      const idx = state.findIndex(i => i.id === action.id);
      state.splice(idx, 1); // direct mutation
      return state;
    default:
      return state;
  }
}`,
    context: 'Reducer must be pure: return a new array without mutating the existing state.',
    hints: ['Use spread or concat for ADD', 'Use filter for REMOVE'],
  },
  {
    id: 'le-005',
    type: 'Logic Error',
    title: 'Async forEach Not Awaited',
    difficulty: 0.5,
    bounty: 45,
    description:
      'An async function uses forEach with an async callback; the outer function resolves before any async operations complete.',
    buggyCode: `async function saveAll(records: Record[]): Promise<void> {
  records.forEach(async (record) => {
    await db.save(record); // never awaited by forEach
  });
  console.log('All saved!'); // fires immediately
}`,
    context: 'saveAll must wait for every db.save to complete before logging.',
    hints: ['Replace forEach with Promise.all + map', 'Or use a for...of loop with await'],
  },
  {
    id: 'le-006',
    type: 'Logic Error',
    title: 'Null Dereference in Chain',
    difficulty: 0.25,
    bounty: 20,
    description:
      'A data transformation chain crashes when an intermediate object is null or undefined.',
    buggyCode: `function getUserCity(user: any): string {
  return user.address.city.toLowerCase();
}`,
    context: 'Should return the city in lowercase, or "unknown" if any part of the chain is missing.',
    hints: ['Use optional chaining (?.)', 'Provide a fallback with ?? "unknown"'],
  },

  // ── Race Condition ────────────────────────────────────────────────────────
  {
    id: 'rc-001',
    type: 'Race Condition',
    title: 'Stale Closure in Async Callback',
    difficulty: 0.6,
    bounty: 60,
    description:
      'A counter increment inside an async callback captures a stale value of count from the closure, causing lost updates.',
    buggyCode: `function Counter() {
  const [count, setCount] = useState(0);

  const handleClick = async () => {
    await delay(100);
    setCount(count + 1); // 'count' is stale
  };

  return <button onClick={handleClick}>{count}</button>;
}`,
    context: 'Rapid clicks should each increment the counter by exactly 1.',
    hints: ['Use the functional form of setState: setCount(c => c + 1)', 'This ensures latest state is used'],
  },
  {
    id: 'rc-002',
    type: 'Race Condition',
    title: 'Unordered Async Responses',
    difficulty: 0.7,
    bounty: 65,
    description:
      'A search input fires a new fetch on every keystroke; slow responses can arrive out of order, displaying stale results.',
    buggyCode: `async function search(query: string) {
  const results = await fetch(\`/api/search?q=\${query}\`).then(r => r.json());
  displayResults(results); // may show results from an older query
}

input.addEventListener('input', (e) => search(e.target.value));`,
    context: 'Only the most recent query\'s results should ever be displayed.',
    hints: ['Use AbortController to cancel previous requests', 'Or track a requestId and ignore stale responses'],
  },
  {
    id: 'rc-003',
    type: 'Race Condition',
    title: 'Double Initialization Guard Missing',
    difficulty: 0.55,
    bounty: 50,
    description:
      'An initialization function can be called concurrently; without a guard, it initializes the resource twice.',
    buggyCode: `let initialized = false;
let db: Database | null = null;

async function initDB(): Promise<Database> {
  if (!initialized) {
    db = await Database.connect();
    initialized = true;
  }
  return db!;
}`,
    context: 'initDB called concurrently should only create one database connection.',
    hints: ['Store the promise itself as the guard', 'Return the same promise to all concurrent callers'],
  },
  {
    id: 'rc-004',
    type: 'Race Condition',
    title: 'Shared Counter Without Lock',
    difficulty: 0.8,
    bounty: 75,
    description:
      'A rate limiter reads and writes a shared counter without synchronization; concurrent requests can bypass the limit.',
    buggyCode: `const requestCounts: Map<string, number> = new Map();

async function rateLimit(userId: string, limit: number): Promise<boolean> {
  const current = requestCounts.get(userId) ?? 0;
  if (current >= limit) return false;
  requestCounts.set(userId, current + 1); // read-modify-write race
  return true;
}`,
    context: 'No more than `limit` concurrent requests should be allowed for a user.',
    hints: ['Use an atomic check-and-increment', 'Or serialize access with a per-user mutex/queue'],
  },
  {
    id: 'rc-005',
    type: 'Race Condition',
    title: 'useEffect Dependency Missing',
    difficulty: 0.45,
    bounty: 40,
    description:
      'A useEffect that fetches data is missing a dependency, so it never re-runs when the relevant prop changes.',
    buggyCode: `function PostDetail({ postId }) {
  const [post, setPost] = useState(null);

  useEffect(() => {
    fetch(\`/api/posts/\${postId}\`)
      .then(r => r.json())
      .then(setPost);
  }, []); // postId missing from deps

  return <div>{post?.title}</div>;
}`,
    context: 'Post detail should always reflect the current postId prop.',
    hints: ['Add postId to the dependency array', 'Consider adding an AbortController cleanup'],
  },

  // ── Buffer Overflow ───────────────────────────────────────────────────────
  {
    id: 'bo-001',
    type: 'Buffer Overflow',
    title: 'Array Index Out of Bounds',
    difficulty: 0.25,
    bounty: 20,
    description:
      'A loop iterates one index beyond the end of an array, causing an undefined read.',
    buggyCode: `function sumArray(arr: number[]): number {
  let total = 0;
  for (let i = 0; i <= arr.length; i++) { // off-by-one
    total += arr[i];
  }
  return total;
}`,
    context: 'sumArray should return the sum of all elements without accessing undefined indices.',
    hints: ['Use strict less-than: i < arr.length', 'Or use arr.reduce'],
  },
  {
    id: 'bo-002',
    type: 'Buffer Overflow',
    title: 'Unbounded String Concatenation',
    difficulty: 0.4,
    bounty: 35,
    description:
      'A log aggregator concatenates strings in a loop without size limits, eventually exhausting available memory.',
    buggyCode: `function aggregateLogs(logLines: string[]): string {
  let result = '';
  for (const line of logLines) {
    result += line + '\\n'; // O(n²) and unbounded
  }
  return result;
}`,
    context: 'Should aggregate logs efficiently with a configurable max size limit.',
    hints: ['Use array.join for O(n) concatenation', 'Add a maxBytes cap and truncate early'],
  },
  {
    id: 'bo-003',
    type: 'Buffer Overflow',
    title: 'Recursive Depth Without Guard',
    difficulty: 0.6,
    bounty: 55,
    description:
      'A tree traversal function has no maximum depth guard; deeply nested or circular structures will overflow the call stack.',
    buggyCode: `function flatten(node: TreeNode): string[] {
  const results = [node.value];
  for (const child of node.children) {
    results.push(...flatten(child)); // no depth limit
  }
  return results;
}`,
    context: 'Flatten should handle deep trees safely without stack overflow.',
    hints: ['Add a depth parameter with a maximum', 'Or convert to an iterative approach with an explicit stack'],
  },
  {
    id: 'bo-004',
    type: 'Buffer Overflow',
    title: 'JSON Parse Without Size Check',
    difficulty: 0.35,
    bounty: 30,
    description:
      'An API endpoint parses incoming JSON without validating payload size, allowing oversized inputs to exhaust memory.',
    buggyCode: `app.post('/api/data', (req, res) => {
  const body = JSON.parse(req.body); // no size validation
  processData(body);
  res.json({ ok: true });
});`,
    context: 'Endpoint should reject payloads above a safe size limit before parsing.',
    hints: ['Check Content-Length header before parsing', 'Use express.json({ limit: "1mb" }) middleware'],
  },
  {
    id: 'bo-005',
    type: 'Buffer Overflow',
    title: 'Infinite Pagination Loop',
    difficulty: 0.5,
    bounty: 45,
    description:
      'A paginated data fetcher has no upper-page guard; if the API always returns results, the loop runs indefinitely.',
    buggyCode: `async function fetchAll(endpoint: string): Promise<Item[]> {
  let page = 0;
  const all: Item[] = [];
  while (true) {
    const batch = await fetchPage(endpoint, page++);
    if (batch.length === 0) break;
    all.push(...batch);
  }
  return all;
}`,
    context: 'fetchAll should collect all pages but abort after a reasonable maximum page count.',
    hints: ['Add a MAX_PAGES constant', 'Break out of the loop when page > MAX_PAGES'],
  },

  // ── API Timeout ───────────────────────────────────────────────────────────
  {
    id: 'at-001',
    type: 'API Timeout',
    title: 'Fetch Without Timeout',
    difficulty: 0.3,
    bounty: 25,
    description:
      'A fetch call has no timeout; if the server hangs, the request waits indefinitely, blocking the UI.',
    buggyCode: `async function loadUserData(userId: string) {
  const res = await fetch(\`/api/users/\${userId}\`);
  return res.json();
}`,
    context: 'loadUserData should resolve within 5 seconds or throw a TimeoutError.',
    hints: ['Use AbortController with setTimeout', 'Or wrap fetch with a Promise.race timeout'],
  },
  {
    id: 'at-002',
    type: 'API Timeout',
    title: 'No Retry on Transient Error',
    difficulty: 0.45,
    bounty: 40,
    description:
      'An API client throws immediately on a 503 response instead of retrying with exponential backoff.',
    buggyCode: `async function callAPI(url: string): Promise<unknown> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(\`HTTP \${res.status}\`);
  return res.json();
}`,
    context: 'On 503 or network errors, should retry up to 3 times with exponential backoff.',
    hints: ['Wrap in a retry loop checking status code', 'Use delay(2 ** attempt * 100) between retries'],
  },
  {
    id: 'at-003',
    type: 'API Timeout',
    title: 'Sequential Requests That Should Be Parallel',
    difficulty: 0.4,
    bounty: 35,
    description:
      'Three independent API calls are made sequentially with await, tripling total latency unnecessarily.',
    buggyCode: `async function loadDashboard(userId: string) {
  const profile = await fetchProfile(userId);
  const posts   = await fetchPosts(userId);   // waits for profile
  const friends = await fetchFriends(userId); // waits for posts
  return { profile, posts, friends };
}`,
    context: 'Dashboard should load all three resources in parallel to minimize latency.',
    hints: ['Use Promise.all to run concurrently', 'Destructure the results array'],
  },
  {
    id: 'at-004',
    type: 'API Timeout',
    title: 'Missing Error Boundary on Fetch',
    difficulty: 0.35,
    bounty: 30,
    description:
      'An async data loader has no try/catch; a failed fetch crashes the entire render tree instead of showing an error state.',
    buggyCode: `async function loadData(): Promise<Data[]> {
  const res = await fetch('/api/data');
  const json = await res.json();
  return json.items;
}`,
    context: 'loadData should catch network and parsing errors and return an empty array with an error log.',
    hints: ['Wrap the entire body in try/catch', 'Return [] and console.error on failure'],
  },
  {
    id: 'at-005',
    type: 'API Timeout',
    title: 'Token Refresh Race on 401',
    difficulty: 0.75,
    bounty: 70,
    description:
      'When multiple requests receive a 401 simultaneously, each one independently triggers a token refresh, causing duplicate refresh calls.',
    buggyCode: `let token = getStoredToken();

async function authFetch(url: string) {
  let res = await fetch(url, { headers: { Authorization: \`Bearer \${token}\` } });
  if (res.status === 401) {
    token = await refreshToken(); // multiple callers can reach here concurrently
    res = await fetch(url, { headers: { Authorization: \`Bearer \${token}\` } });
  }
  return res.json();
}`,
    context: 'Token refresh should happen exactly once per expiry; concurrent 401s should queue on the single refresh promise.',
    hints: ['Cache the in-flight refresh promise', 'All concurrent callers await the same promise'],
  },
];

/**
 * Get a random challenge for a given error node type.
 */
export function getChallengeForType(type: ErrorNodeType): Challenge {
  const matching = CHALLENGES.filter(c => c.type === type);
  if (matching.length === 0) return CHALLENGES[0];
  return matching[Math.floor(Math.random() * matching.length)];
}

/**
 * Get a specific challenge by ID.
 */
export function getChallengeById(id: string): Challenge | undefined {
  return CHALLENGES.find(c => c.id === id);
}
