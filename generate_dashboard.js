const fs = require('fs');
const d = fs.readFileSync('d:/inspire_lms/routes.csv', 'utf8');
const rows = d.split('\n').filter(r => r && !r.startsWith('route,')).map(r => r.split(','));

const html = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Inspire LMS - Route Dashboard</title>
<style>
  body { font-family: "Inter", sans-serif; background: #0f172a; color: #f8fafc; padding: 2rem; }
  a { color: #38bdf8; text-decoration: none; }
  a:hover { text-decoration: underline; }
  table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
  th, td { border: 1px solid #334155; padding: 0.75rem; text-align: left; }
  th { background: #1e293b; }
  .access-Public { color: #a3e635; }
  .access-admin { color: #f43f5e; }
  .access-teacher { color: #fbbf24; }
  .access-student { color: #60a5fa; }
  .btn { display: inline-block; padding: 0.5rem 1rem; background: #3b82f6; color: white; border-radius: 4px; margin-right: 1rem; cursor: pointer; }
  .btn:hover { background: #2563eb; color: white; text-decoration: none; }
  .flex { display: flex; gap: 40px; margin-bottom: 20px; align-items: start; }
  .instructions { background: #1e293b; padding: 1.5rem; border-radius: 8px; border-left: 4px solid #3b82f6; }
</style>
</head>
<body>
<h1>Inspire LMS - Route Testing Dashboard</h1>

<div class="flex">
  <div class="instructions">
    <h2>1. Login Accounts</h2>
    <p>Please log in to http://localhost:3000 using one of these accounts:</p>
    <ul>
      <li><strong>Admin:</strong> admin@example.com / Password123!</li>
      <li><strong>Teacher:</strong> teacher@example.com / Password123!</li>
      <li><strong>Student:</strong> student@example.com / Password123!</li>
    </ul>
    <p><em>(These active accounts were created successfully in the Postgres database.)</em></p>
  </div>
  <div class="instructions">
    <h2>2. Open All Routes</h2>
    <p>Using the button below will attempt to open all 69 routes locally!<br>
    Please ensure your browser POPUPS ARE ALLOWED.</p>
    <button onclick="openAll()" style="padding:10px 20px;background:#10b981;color:white;border:none;border-radius:5px;cursor:pointer;font-weight:bold;font-size:16px;">
      🚀 Auto-Open All Routes in Background Tabs
    </button>
  </div>
</div>

<table>
  <tr>
    <th>Route</th>
    <th>Component</th>
    <th>Access Gate</th>
    <th>Testing Action</th>
  </tr>
  ${rows.map(r => {
    if(!r[0]) return '';
    let url = r[0]
      .replace('/:courseId', '/demo-course-123')
      .replace('/:id', '/123')
      .replace('/:sessionId', '/sess-123')
      .replace('/:assignmentId', '/asn-123')
      .replace('/:quizId', '/quiz-123')
      .replace('/:attemptId', '/att-123')
      .replace('/:submissionId', '/sub-123')
      .replace('/:bookId', '/book-123')
      .replace('/:orderId', '/ord-123')
      .replace('/:itemId', '/item-123')
      .replace('/:studentId', '/stu-123')
      .replace('/:itemType', '/course');
    
    const acs = (r[2] || '').split('|')[0] || '';
    
    return `<tr>
      <td><code>${r[0]}</code></td>
      <td>${r[1]}</td>
      <td class="access-${acs}"><strong>${r[2]}</strong></td>
      <td>
        <a href="http://localhost:3000${url.replace('*', 'does-not-exist')}" target="_blank" class="btn test-btn">Open Page</a>
      </td>
    </tr>`;
  }).join('')}
</table>

<script>
function openAll() {
  const links = document.querySelectorAll('.test-btn');
  if(!confirm('Are you sure? This will open ' + links.length + ' tabs.')) return;
  links.forEach((l, i) => {
    setTimeout(() => {
      window.open(l.href, '_blank');
    }, i * 300); // Stagger by 300ms
  });
}
</script>
</body>
</html>
`;

fs.writeFileSync('d:/inspire_lms/routes_launcher.html', html);
console.log('Dashboard generated.');
