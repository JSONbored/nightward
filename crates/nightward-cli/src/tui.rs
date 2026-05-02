use anyhow::Result;
use nightward_core::{max_risk, Report, RiskLevel};
use opentui::buffer::{BoxOptions, BoxStyle, ClipRect, TitleAlign};
use opentui::input::{Event, InputParser, KeyCode};
use opentui::terminal::{enable_raw_mode, terminal_size};
use opentui::{OptimizedBuffer, Renderer, Rgba, Style};
use opentui_rust as opentui;
use std::io::{self, Read};
use std::sync::mpsc;
use std::time::Duration;

const VIEWS: [&str; 7] = [
    "Dashboard",
    "Findings",
    "Analysis",
    "Fix Plan",
    "History",
    "Providers",
    "Help",
];

pub fn run(report: &Report) -> Result<()> {
    let (term_w, term_h) = terminal_size().unwrap_or((120, 36));
    let mut renderer = Renderer::new(u32::from(term_w), u32::from(term_h))?;
    let _raw_guard = enable_raw_mode()?;
    let mut app = TuiState::new(report);
    let mut parser = InputParser::new();
    let (tx, rx) = mpsc::channel::<Vec<u8>>();

    let _input_thread = std::thread::spawn(move || {
        let mut stdin = io::stdin();
        let mut buf = [0u8; 64];
        loop {
            match stdin.read(&mut buf) {
                Ok(0) => {}
                Ok(n) => {
                    if tx.send(buf[..n].to_vec()).is_err() {
                        break;
                    }
                }
                Err(ref err) if err.kind() == io::ErrorKind::WouldBlock => {}
                Err(_) => break,
            }
        }
    });

    loop {
        let (width, height) = renderer.size();
        app.render(renderer.buffer(), width, height);
        renderer.present()?;

        if std::env::var("NIGHTWARD_TUI_CAPTURE").as_deref() == Ok("1") {
            break;
        }

        let mut keep_running = true;
        for chunk in rx.try_iter() {
            let mut offset = 0;
            while offset < chunk.len() {
                let Ok((event, used)) = parser.parse(&chunk[offset..]) else {
                    break;
                };
                offset += used;
                if let Event::Resize(resize) = &event {
                    renderer.resize(u32::from(resize.width), u32::from(resize.height))?;
                }
                if !app.handle_event(&event) {
                    keep_running = false;
                    break;
                }
            }
            if !keep_running {
                break;
            }
        }
        if !keep_running {
            break;
        }
        app.tick();
        std::thread::sleep(Duration::from_millis(33));
    }

    Ok(())
}

struct TuiState<'a> {
    report: &'a Report,
    active_view: usize,
    selected_finding: usize,
    frame: u64,
    palette: Palette,
}

impl<'a> TuiState<'a> {
    fn new(report: &'a Report) -> Self {
        Self {
            report,
            active_view: 0,
            selected_finding: 0,
            frame: 0,
            palette: Palette::new(),
        }
    }

    fn tick(&mut self) {
        self.frame = self.frame.wrapping_add(1);
    }

    fn handle_event(&mut self, event: &Event) -> bool {
        let Event::Key(key) = event else {
            return true;
        };
        if key.is_ctrl_c() || matches!(key.code, KeyCode::Char('q') | KeyCode::Esc) {
            return false;
        }
        match key.code {
            KeyCode::Tab => self.active_view = (self.active_view + 1) % VIEWS.len(),
            KeyCode::BackTab => {
                self.active_view = self.active_view.checked_sub(1).unwrap_or(VIEWS.len() - 1);
            }
            KeyCode::Char(ch @ '1'..='7') => {
                self.active_view = usize::from(ch as u8 - b'1');
            }
            KeyCode::Down | KeyCode::Char('j') => self.select_next_finding(),
            KeyCode::Up | KeyCode::Char('k') => self.select_previous_finding(),
            _ => {}
        }
        true
    }

    fn select_next_finding(&mut self) {
        if self.report.findings.is_empty() {
            self.selected_finding = 0;
            return;
        }
        self.selected_finding = (self.selected_finding + 1) % self.report.findings.len();
    }

    fn select_previous_finding(&mut self) {
        if self.report.findings.is_empty() {
            self.selected_finding = 0;
            return;
        }
        self.selected_finding = self
            .selected_finding
            .checked_sub(1)
            .unwrap_or(self.report.findings.len() - 1);
    }

    fn render(&self, buffer: &mut OptimizedBuffer, width: u32, height: u32) {
        buffer.clear(self.palette.bg);
        if width < 72 || height < 22 {
            self.render_tiny(buffer, width, height);
            return;
        }

        self.render_header(buffer, width);
        self.render_footer(buffer, width, height);

        let content_y = 4;
        let content_h = height.saturating_sub(6);
        let sidebar_w = (width / 5).clamp(22, 30);
        let main_x = sidebar_w + 1;
        let main_w = width.saturating_sub(main_x + 1);

        self.render_sidebar(
            buffer,
            Area::new(1, content_y, sidebar_w.saturating_sub(1), content_h),
        );

        let top_h = 7.min(content_h / 3).max(5);
        let lower_y = content_y + top_h + 1;
        let lower_h = content_h.saturating_sub(top_h + 1);
        self.render_metrics(buffer, Area::new(main_x, content_y, main_w, top_h));

        let list_w = (main_w * 45 / 100).clamp(32, main_w.saturating_sub(36));
        let detail_x = main_x + list_w + 1;
        let detail_w = main_w.saturating_sub(list_w + 1);
        self.render_findings(buffer, Area::new(main_x, lower_y, list_w, lower_h));
        self.render_detail(buffer, Area::new(detail_x, lower_y, detail_w, lower_h));
    }

    fn render_tiny(&self, buffer: &mut OptimizedBuffer, width: u32, height: u32) {
        buffer.clear(self.palette.bg);
        let title = "Nightward";
        draw_text(
            buffer,
            2,
            1,
            title,
            Style::fg(self.palette.cyan).with_bold(),
        );
        draw_text(
            buffer,
            2,
            3,
            &status_label(
                max_risk(&self.report.findings),
                self.report.summary.total_findings,
            ),
            Style::fg(severity_color(
                &self.palette,
                max_risk(&self.report.findings),
            ))
            .with_bold(),
        );
        if height > 6 && width > 16 {
            draw_text(
                buffer,
                2,
                5,
                "Grow the terminal for the full dashboard.",
                Style::fg(self.palette.muted),
            );
        }
    }

    fn render_header(&self, buffer: &mut OptimizedBuffer, width: u32) {
        buffer.fill_rect(0, 0, width, 3, self.palette.header);
        draw_text(
            buffer,
            2,
            0,
            "Nightward",
            Style::fg(self.palette.cyan).with_bold(),
        );
        draw_text(
            buffer,
            13,
            0,
            "Audit AI tool configs before they leak, drift, or sync badly.",
            Style::fg(self.palette.white),
        );
        draw_text(
            buffer,
            2,
            1,
            "local-first  no telemetry  read-only scans  redacted evidence",
            Style::fg(self.palette.muted),
        );

        let status = compact_status_label(self.report);
        let status_w = text_width(&status).saturating_add(6).min(30);
        let status_x = width.saturating_sub(status_w + 2);
        draw_chip(
            buffer,
            status_x,
            1,
            status_w,
            &status,
            severity_color(&self.palette, max_risk(&self.report.findings)),
            self.palette.bg,
        );
    }

    fn render_footer(&self, buffer: &mut OptimizedBuffer, width: u32, height: u32) {
        let y = height.saturating_sub(2);
        buffer.fill_rect(0, y, width, 2, self.palette.header);
        let hints = [
            ("1-7", "views"),
            ("j/k", "move"),
            ("tab", "next view"),
            ("/", "filter"),
            ("e", "export plan"),
            ("q", "quit"),
        ];
        let mut x = 2;
        for (key, label) in hints {
            draw_text(buffer, x, y, key, Style::fg(self.palette.cyan).with_bold());
            x += text_width(key) + 1;
            draw_text(buffer, x, y, label, Style::fg(self.palette.muted));
            x += text_width(label) + 3;
        }
        let version = format!("schema v{}", self.report.schema_version);
        draw_text(
            buffer,
            width.saturating_sub(text_width(&version) + 2),
            y,
            &version,
            Style::fg(self.palette.muted),
        );
    }

    fn render_sidebar(&self, buffer: &mut OptimizedBuffer, area: Area) {
        draw_panel(
            buffer,
            area,
            "Command Center",
            self.palette.cyan,
            self.palette.panel,
        );
        let mut row = area.y + 2;
        for (idx, view) in VIEWS.iter().enumerate() {
            let active = idx == self.active_view;
            let color = view_color(&self.palette, idx);
            if active {
                buffer.fill_rect(
                    area.x + 1,
                    row,
                    area.w.saturating_sub(2),
                    1,
                    color.with_alpha(0.18),
                );
            }
            draw_text(
                buffer,
                area.x + 2,
                row,
                if active { "▶" } else { "•" },
                Style::fg(color).with_bold(),
            );
            draw_text(
                buffer,
                area.x + 5,
                row,
                view,
                if active {
                    Style::fg(self.palette.white).with_bold()
                } else {
                    Style::fg(self.palette.muted)
                },
            );
            row += 2;
        }

        let risk = max_risk(&self.report.findings);
        let posture_y = area.y + area.h.saturating_sub(8);
        draw_text(
            buffer,
            area.x + 2,
            posture_y,
            "Posture",
            Style::fg(self.palette.muted),
        );
        draw_text(
            buffer,
            area.x + 2,
            posture_y + 1,
            &status_label(risk, self.report.summary.total_findings),
            Style::fg(severity_color(&self.palette, risk)).with_bold(),
        );
        draw_text(
            buffer,
            area.x + 2,
            posture_y + 3,
            "Plan-only remediation",
            Style::fg(self.palette.green),
        );
        draw_text(
            buffer,
            area.x + 2,
            posture_y + 4,
            "No config writes",
            Style::fg(self.palette.green),
        );
    }

    fn render_metrics(&self, buffer: &mut OptimizedBuffer, area: Area) {
        let gap = 1;
        let card_w = area.w.saturating_sub(gap * 3) / 4;
        let metrics = [
            ("Items", self.report.summary.total_items, self.palette.cyan),
            (
                "Findings",
                self.report.summary.total_findings,
                severity_color(&self.palette, max_risk(&self.report.findings)),
            ),
            (
                "Critical",
                count_severity(self.report, RiskLevel::Critical),
                self.palette.red,
            ),
            (
                "High",
                count_severity(self.report, RiskLevel::High),
                self.palette.orange,
            ),
        ];
        for (idx, (label, value, color)) in metrics.iter().enumerate() {
            let cx = area.x + u32::try_from(idx).unwrap_or(0) * (card_w + gap);
            metric_card(
                buffer,
                Area::new(cx, area.y, card_w, area.h),
                label,
                *value,
                *color,
                &self.palette,
            );
        }
    }

    fn render_findings(&self, buffer: &mut OptimizedBuffer, area: Area) {
        draw_panel(
            buffer,
            area,
            "Findings",
            severity_color(&self.palette, max_risk(&self.report.findings)),
            self.palette.panel,
        );
        buffer.push_scissor(ClipRect::new(
            i32::try_from(area.x + 1).unwrap_or(0),
            i32::try_from(area.y + 1).unwrap_or(0),
            area.w.saturating_sub(2),
            area.h.saturating_sub(2),
        ));

        draw_text(
            buffer,
            area.x + 2,
            area.y + 2,
            "SEV",
            Style::fg(self.palette.muted),
        );
        draw_text(
            buffer,
            area.x + 8,
            area.y + 2,
            "RULE",
            Style::fg(self.palette.muted),
        );
        draw_text(
            buffer,
            area.x + area.w.saturating_sub(14),
            area.y + 2,
            "SERVER",
            Style::fg(self.palette.muted),
        );

        let max_rows = area.h.saturating_sub(5) as usize;
        for (idx, finding) in self.report.findings.iter().take(max_rows).enumerate() {
            let row = area.y + 4 + u32::try_from(idx).unwrap_or(0);
            let selected = idx == self.selected_finding;
            let color = severity_color(&self.palette, finding.severity);
            if selected {
                buffer.fill_rect(
                    area.x + 1,
                    row,
                    area.w.saturating_sub(2),
                    1,
                    color.with_alpha(0.20),
                );
            }
            draw_text(
                buffer,
                area.x + 2,
                row,
                severity_badge(finding.severity),
                Style::fg(color).with_bold(),
            );
            draw_text(
                buffer,
                area.x + 8,
                row,
                &truncate(&finding.rule, area.w.saturating_sub(25) as usize),
                Style::fg(if selected {
                    self.palette.white
                } else {
                    self.palette.muted
                }),
            );
            draw_text(
                buffer,
                area.x + area.w.saturating_sub(14),
                row,
                &truncate(&finding.server, 12),
                Style::fg(self.palette.white),
            );
        }
        buffer.pop_scissor();
    }

    fn render_detail(&self, buffer: &mut OptimizedBuffer, area: Area) {
        draw_panel(
            buffer,
            area,
            "Evidence Review",
            self.palette.green,
            self.palette.panel,
        );
        let inner_x = area.x + 2;
        let mut row = area.y + 2;
        let inner_w = area.w.saturating_sub(4) as usize;

        if let Some(finding) = self.report.findings.get(self.selected_finding) {
            let severity = severity_color(&self.palette, finding.severity);
            draw_chip(
                buffer,
                inner_x,
                row,
                10,
                severity_badge(finding.severity),
                severity,
                self.palette.panel,
            );
            draw_text(
                buffer,
                inner_x + 12,
                row,
                &truncate(&finding.rule, inner_w.saturating_sub(12)),
                Style::fg(severity).with_bold(),
            );
            row += 2;

            for line in wrap(&finding.message, inner_w) {
                draw_text(buffer, inner_x, row, &line, Style::fg(self.palette.white));
                row += 1;
            }
            row += 1;

            section_label(buffer, inner_x, row, "Evidence", self.palette.cyan);
            row += 1;
            for line in wrap(&finding.evidence, inner_w).into_iter().take(4) {
                draw_text(
                    buffer,
                    inner_x,
                    row,
                    &line,
                    Style::fg(self.palette.muted).with_bg(self.palette.code_bg),
                );
                row += 1;
            }
            row += 1;

            section_label(buffer, inner_x, row, "Why it matters", self.palette.amber);
            row += 1;
            for line in wrap(&finding.why, inner_w).into_iter().take(5) {
                draw_text(buffer, inner_x, row, &line, Style::fg(self.palette.muted));
                row += 1;
            }

            let action_y = area.y + area.h.saturating_sub(7);
            draw_panel(
                buffer,
                Area::new(inner_x, action_y, area.w.saturating_sub(4), 5),
                "Next Action",
                self.palette.amber,
                self.palette.surface,
            );
            for (idx, line) in wrap(&finding.recommended_action, inner_w.saturating_sub(2))
                .into_iter()
                .take(2)
                .enumerate()
            {
                draw_text(
                    buffer,
                    inner_x + 2,
                    action_y + 2 + u32::try_from(idx).unwrap_or(0),
                    &line,
                    Style::fg(self.palette.white),
                );
            }
            draw_text(
                buffer,
                inner_x + 2,
                action_y + 4,
                "nw fix plan --all --json",
                Style::fg(self.palette.green).with_bold(),
            );
        } else {
            draw_text(
                buffer,
                inner_x,
                row,
                "No findings in this scan.",
                Style::fg(self.palette.green).with_bold(),
            );
            draw_text(
                buffer,
                inner_x,
                row + 2,
                "Keep this report as the clean baseline for future diffs.",
                Style::fg(self.palette.muted),
            );
        }
    }
}

#[derive(Clone, Copy)]
struct Palette {
    bg: Rgba,
    header: Rgba,
    panel: Rgba,
    surface: Rgba,
    code_bg: Rgba,
    white: Rgba,
    muted: Rgba,
    cyan: Rgba,
    green: Rgba,
    amber: Rgba,
    red: Rgba,
    orange: Rgba,
    magenta: Rgba,
    blue: Rgba,
}

impl Palette {
    fn new() -> Self {
        Self {
            bg: color("#090b12"),
            header: color("#101624"),
            panel: color("#111827"),
            surface: color("#172033"),
            code_bg: color("#0b1020"),
            white: color("#f8fafc"),
            muted: color("#8b93ad"),
            cyan: color("#67e8f9"),
            green: color("#5eead4"),
            amber: color("#facc15"),
            red: color("#fb7185"),
            orange: color("#fb923c"),
            magenta: color("#e879f9"),
            blue: color("#60a5fa"),
        }
    }
}

#[derive(Clone, Copy)]
struct Area {
    x: u32,
    y: u32,
    w: u32,
    h: u32,
}

impl Area {
    const fn new(x: u32, y: u32, w: u32, h: u32) -> Self {
        Self { x, y, w, h }
    }
}

fn draw_panel(buffer: &mut OptimizedBuffer, area: Area, title: &str, border: Rgba, fill: Rgba) {
    if area.w < 4 || area.h < 3 {
        return;
    }
    let mut options = BoxOptions::new(BoxStyle::rounded(Style::fg(border)));
    options.fill = Some(fill.with_alpha(0.92));
    options.title = Some(format!(" {title} "));
    options.title_align = TitleAlign::Left;
    buffer.draw_box_with_options(area.x, area.y, area.w, area.h, options);
}

fn metric_card(
    buffer: &mut OptimizedBuffer,
    area: Area,
    label: &str,
    value: usize,
    color: Rgba,
    palette: &Palette,
) {
    draw_panel(buffer, area, label, color, palette.panel);
    let value_s = value.to_string();
    draw_text(
        buffer,
        area.x + 2,
        area.y + 2,
        &value_s,
        Style::fg(color).with_bold(),
    );
    draw_text(
        buffer,
        area.x + 2,
        area.y + 3,
        match label {
            "Items" => "tracked configs",
            "Findings" => "review items",
            "Critical" => "fix first",
            _ => "needs pinning",
        },
        Style::fg(palette.muted),
    );
    let bar_w = area.w.saturating_sub(4);
    if area.h > 5 && bar_w > 4 {
        let filled = if value == 0 {
            1
        } else {
            (u32::try_from(value.min(12)).unwrap_or(1) * bar_w / 12).max(1)
        };
        buffer.fill_rect(area.x + 2, area.y + area.h - 2, bar_w, 1, palette.surface);
        buffer.fill_rect(
            area.x + 2,
            area.y + area.h - 2,
            filled.min(bar_w),
            1,
            color.with_alpha(0.55),
        );
    }
}

fn draw_chip(
    buffer: &mut OptimizedBuffer,
    x: u32,
    y: u32,
    w: u32,
    label: &str,
    fg: Rgba,
    bg: Rgba,
) {
    if w < 3 {
        return;
    }
    buffer.fill_rect(x, y, w, 1, fg.with_alpha(0.18));
    let text = truncate(label, w.saturating_sub(2) as usize);
    draw_text(
        buffer,
        x + 1,
        y,
        &text,
        Style::fg(fg).with_bg(bg).with_bold(),
    );
}

fn section_label(buffer: &mut OptimizedBuffer, x: u32, y: u32, label: &str, color: Rgba) {
    draw_text(buffer, x, y, label, Style::fg(color).with_bold());
}

fn draw_text(buffer: &mut OptimizedBuffer, x: u32, y: u32, text: &str, style: Style) {
    if x >= buffer.width() || y >= buffer.height() {
        return;
    }
    buffer.draw_text(x, y, text, style);
}

fn view_color(palette: &Palette, idx: usize) -> Rgba {
    match idx {
        0 => palette.cyan,
        1 => palette.red,
        2 => palette.magenta,
        3 => palette.amber,
        4 => palette.blue,
        5 => palette.green,
        _ => palette.muted,
    }
}

fn severity_color(palette: &Palette, risk: RiskLevel) -> Rgba {
    match risk {
        RiskLevel::Critical => palette.red,
        RiskLevel::High => palette.orange,
        RiskLevel::Medium => palette.amber,
        RiskLevel::Low => palette.blue,
        RiskLevel::Info => palette.muted,
    }
}

fn severity_badge(risk: RiskLevel) -> &'static str {
    match risk {
        RiskLevel::Critical => "CRIT",
        RiskLevel::High => "HIGH",
        RiskLevel::Medium => "MED",
        RiskLevel::Low => "LOW",
        RiskLevel::Info => "INFO",
    }
}

fn compact_status_label(report: &Report) -> String {
    let critical = count_severity(report, RiskLevel::Critical);
    let high = count_severity(report, RiskLevel::High);
    let medium = count_severity(report, RiskLevel::Medium);
    if critical > 0 {
        return format!("{critical}C {high}H / {}", report.summary.total_findings);
    }
    if high > 0 {
        return format!("{high}H {medium}M / {}", report.summary.total_findings);
    }
    if medium > 0 {
        return format!("{medium}M / {}", report.summary.total_findings);
    }
    "OK".to_string()
}

fn status_label(risk: RiskLevel, total: usize) -> String {
    match risk {
        RiskLevel::Critical => format!("{total} findings / critical"),
        RiskLevel::High => format!("{total} findings / high"),
        RiskLevel::Medium => format!("{total} findings / medium"),
        _ => "OK".to_string(),
    }
}

fn count_severity(report: &Report, risk: RiskLevel) -> usize {
    report
        .summary
        .findings_by_severity
        .get(&risk)
        .copied()
        .unwrap_or_default()
}

fn truncate(text: &str, max: usize) -> String {
    if max < 2 {
        return String::new();
    }
    if text.chars().count() <= max {
        return text.to_string();
    }
    let mut out: String = text.chars().take(max.saturating_sub(1)).collect();
    out.push('…');
    out
}

fn wrap(text: &str, max_width: usize) -> Vec<String> {
    if max_width < 8 {
        return vec![truncate(text, max_width)];
    }
    let mut lines = Vec::new();
    let mut current = String::new();
    for word in text.split_whitespace() {
        let candidate_len = current.chars().count() + usize::from(!current.is_empty()) + word.len();
        if candidate_len > max_width && !current.is_empty() {
            lines.push(current);
            current = String::new();
        }
        if !current.is_empty() {
            current.push(' ');
        }
        current.push_str(word);
    }
    if !current.is_empty() {
        lines.push(current);
    }
    if lines.is_empty() {
        lines.push(String::new());
    }
    lines
}

fn text_width(text: &str) -> u32 {
    u32::try_from(text.chars().count()).unwrap_or(u32::MAX)
}

fn color(hex: &str) -> Rgba {
    Rgba::from_hex(hex).expect("valid hard-coded color")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn severity_labels_are_short() {
        assert_eq!(severity_badge(RiskLevel::Critical), "CRIT");
        assert_eq!(severity_badge(RiskLevel::Info), "INFO");
    }

    #[test]
    fn wraps_long_text_without_dropping_words() {
        let lines = wrap("one two three four", 8);
        assert_eq!(lines, vec!["one two", "three", "four"]);
    }
}
