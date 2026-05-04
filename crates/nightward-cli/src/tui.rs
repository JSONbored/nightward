use anyhow::Result;
use nightward_core::analysis::{self, Options as AnalysisOptions};
use nightward_core::fixplan::{self, Selector};
use nightward_core::{backupplan, max_risk, Classification, Finding, Report, RiskLevel};
use opentui::buffer::{BoxOptions, BoxStyle, ClipRect, TitleAlign};
use opentui::input::{Event, InputParser, KeyCode};
use opentui::terminal::{enable_raw_mode, terminal_size};
use opentui::{OptimizedBuffer, Renderer, Rgba, Style};
use opentui_rust as opentui;
use std::io::{self, Read};
use std::sync::mpsc;
use std::time::Duration;

const VIEWS: [&str; 7] = [
    "Overview",
    "Findings",
    "Analysis",
    "Fix Plan",
    "Inventory",
    "Backup",
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
    severity_filter: Option<RiskLevel>,
    search_query: String,
    search_mode: bool,
    frame: u64,
    palette: Palette,
}

impl<'a> TuiState<'a> {
    fn new(report: &'a Report) -> Self {
        Self {
            report,
            active_view: 0,
            selected_finding: 0,
            severity_filter: None,
            search_query: String::new(),
            search_mode: false,
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
        if key.is_ctrl_c() {
            return false;
        }
        if self.search_mode {
            match key.code {
                KeyCode::Esc | KeyCode::Enter => self.search_mode = false,
                KeyCode::Backspace => {
                    self.search_query.pop();
                    self.clamp_selected_finding();
                }
                KeyCode::Char(ch) if !key.ctrl() => {
                    self.search_query.push(ch);
                    self.selected_finding = 0;
                }
                _ => {}
            }
            return true;
        }
        if matches!(key.code, KeyCode::Char('q') | KeyCode::Esc) {
            return false;
        }
        match key.code {
            KeyCode::Tab | KeyCode::Right | KeyCode::Char('l') => self.next_view(),
            KeyCode::BackTab | KeyCode::Left | KeyCode::Char('h') => self.previous_view(),
            KeyCode::Char(ch @ '1'..='7') => {
                self.active_view = usize::from(ch as u8 - b'1');
            }
            KeyCode::Down | KeyCode::Char('j') => self.select_next_finding(),
            KeyCode::Up | KeyCode::Char('k') => self.select_previous_finding(),
            KeyCode::Char('/') => self.search_mode = true,
            KeyCode::Char('s') => self.cycle_severity_filter(),
            KeyCode::Char('x') => self.clear_filters(),
            _ => {}
        }
        true
    }

    fn next_view(&mut self) {
        self.active_view = (self.active_view + 1) % VIEWS.len();
    }

    fn previous_view(&mut self) {
        self.active_view = self.active_view.checked_sub(1).unwrap_or(VIEWS.len() - 1);
    }

    fn select_next_finding(&mut self) {
        let count = self.display_findings().len();
        if count == 0 {
            self.selected_finding = 0;
            return;
        }
        self.selected_finding = (self.selected_finding + 1) % count;
    }

    fn select_previous_finding(&mut self) {
        let count = self.display_findings().len();
        if count == 0 {
            self.selected_finding = 0;
            return;
        }
        self.selected_finding = self.selected_finding.checked_sub(1).unwrap_or(count - 1);
    }

    fn clamp_selected_finding(&mut self) {
        let count = self.display_findings().len();
        if count == 0 {
            self.selected_finding = 0;
        } else if self.selected_finding >= count {
            self.selected_finding = count - 1;
        }
    }

    fn cycle_severity_filter(&mut self) {
        self.severity_filter = match self.severity_filter {
            None => Some(RiskLevel::Critical),
            Some(RiskLevel::Critical) => Some(RiskLevel::High),
            Some(RiskLevel::High) => Some(RiskLevel::Medium),
            Some(RiskLevel::Medium) => Some(RiskLevel::Low),
            Some(RiskLevel::Low) => Some(RiskLevel::Info),
            Some(RiskLevel::Info) => None,
        };
        self.selected_finding = 0;
    }

    fn clear_filters(&mut self) {
        self.severity_filter = None;
        self.search_query.clear();
        self.search_mode = false;
        self.selected_finding = 0;
    }

    fn render(&self, buffer: &mut OptimizedBuffer, width: u32, height: u32) {
        buffer.clear(self.palette.bg);
        if width < 72 || height < 18 {
            self.render_tiny(buffer, width, height);
            return;
        }
        if width < 96 || height < 26 {
            self.render_compact(buffer, width, height);
            return;
        }

        let sidebar_w = 30.min(width.saturating_sub(74));
        let main_x = sidebar_w + 2;
        let main_w = width.saturating_sub(main_x + 2);
        self.render_sidebar(buffer, Area::new(0, 0, sidebar_w, height));
        self.render_header(buffer, Area::new(main_x, 1, main_w, 6));
        self.render_content(
            buffer,
            Area::new(main_x, 8, main_w, height.saturating_sub(11)),
        );
        if self.active_view == 0 {
            draw_hline(
                buffer,
                main_x,
                height.saturating_sub(3),
                main_w,
                self.palette.line,
            );
        } else {
            self.render_footer(
                buffer,
                Area::new(main_x, height.saturating_sub(3), main_w, 2),
            );
        }
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

    fn render_compact(&self, buffer: &mut OptimizedBuffer, width: u32, height: u32) {
        buffer.clear(self.palette.bg);
        draw_text(
            buffer,
            1,
            1,
            "NIGHTWARD",
            Style::fg(self.palette.cyan).with_bold(),
        );
        draw_text(
            buffer,
            12,
            1,
            "AI config risk console",
            Style::fg(self.palette.muted),
        );
        let risk = max_risk(&self.report.findings);
        draw_text(
            buffer,
            width.saturating_sub(20),
            1,
            &status_label(risk, self.report.summary.total_findings),
            Style::fg(severity_color(&self.palette, risk)).with_bold(),
        );
        draw_text(
            buffer,
            1,
            3,
            &format!(
                "{}  severity {}  search {}",
                VIEWS[self.active_view],
                severity_filter_label(self.severity_filter),
                search_label(&self.search_query, self.search_mode)
            ),
            Style::fg(view_nav_color(&self.palette, self.active_view)).with_bold(),
        );
        draw_hline(buffer, 1, 4, width.saturating_sub(2), self.palette.line);

        let nav = "1 Overview  2 Findings  3 Analysis  4 Fix Plan  5 Inventory  6 Backup  7 Help";
        draw_text(
            buffer,
            1,
            height.saturating_sub(2),
            &truncate(nav, width.saturating_sub(2) as usize),
            Style::fg(self.palette.muted),
        );
        let body = Area::new(1, 6, width.saturating_sub(2), height.saturating_sub(10));
        self.render_content(buffer, body);
    }

    fn render_header(&self, buffer: &mut OptimizedBuffer, area: Area) {
        draw_text(
            buffer,
            area.x,
            area.y,
            "Review what AI tools can read, run, and accidentally sync.",
            Style::fg(self.palette.white).with_bold(),
        );
        draw_text(
            buffer,
            area.x,
            area.y + 1,
            &format!(
                "generated {}",
                self.report.generated_at.format("%-m/%-d/%Y, %-I:%M:%S %p")
            ),
            Style::fg(self.palette.muted),
        );

        let card_w = area.w.saturating_sub(3) / 4;
        let cards = [
            (
                "findings",
                self.report.summary.total_findings.to_string(),
                severity_color(&self.palette, max_risk(&self.report.findings)),
            ),
            (
                "items",
                self.report.summary.total_items.to_string(),
                self.palette.blue,
            ),
            (
                "mode",
                if self.report.scan_mode.is_empty() {
                    if self.report.workspace.is_empty() {
                        "home".to_string()
                    } else {
                        "workspace".to_string()
                    }
                } else {
                    self.report.scan_mode.clone()
                },
                self.palette.magenta,
            ),
            (
                "active",
                VIEWS[self.active_view].to_string(),
                self.palette.cyan,
            ),
        ];
        for (idx, (label, value, color)) in cards.iter().enumerate() {
            let x = area.x + u32::try_from(idx).unwrap_or(0) * (card_w + 1);
            old_stat_card(
                buffer,
                Area::new(x, area.y + 2, card_w, 4),
                label,
                value,
                *color,
                &self.palette,
            );
        }
        draw_hline(buffer, area.x, area.y + area.h, area.w, self.palette.line);
    }

    fn render_footer(&self, buffer: &mut OptimizedBuffer, area: Area) {
        draw_hline(buffer, area.x, area.y, area.w, self.palette.line);
        let y = area.y + 1;
        let hints = [
            ("tab/1-7", "navigate"),
            ("/", "search"),
            ("s", "severity"),
            ("x", "clear"),
            ("q", "quit"),
        ];
        let mut x = area.x;
        for (key, label) in hints {
            draw_text(buffer, x, y, key, Style::fg(self.palette.cyan).with_bold());
            x += text_width(key) + 1;
            draw_text(buffer, x, y, label, Style::fg(self.palette.muted));
            x += text_width(label) + 3;
        }
    }

    fn render_sidebar(&self, buffer: &mut OptimizedBuffer, area: Area) {
        buffer.fill_rect(area.x, area.y, area.w, area.h, self.palette.sidebar);
        draw_vline(
            buffer,
            area.x + area.w.saturating_sub(1),
            area.y,
            area.h,
            self.palette.line,
        );
        let x = area.x + 1;
        let mut row = area.y + 2;
        draw_text(
            buffer,
            x,
            row,
            "NIGHTWARD",
            Style::fg(self.palette.cyan).with_bold(),
        );
        row += 1;
        draw_text(
            buffer,
            x,
            row,
            "AI config risk console",
            Style::fg(self.palette.muted),
        );
        row += 2;

        let risk = max_risk(&self.report.findings);
        plain_box(
            buffer,
            Area::new(x, row, area.w.saturating_sub(6), 3),
            severity_color(&self.palette, risk),
            self.palette.sidebar,
        );
        draw_text(
            buffer,
            x + 2,
            row + 1,
            &format!("RISK {}", risk_label(risk)),
            Style::fg(severity_color(&self.palette, risk)).with_bold(),
        );
        row += 3;
        draw_text(
            buffer,
            x,
            row,
            &format!("{} findings", self.report.summary.total_findings),
            Style::fg(self.palette.white),
        );
        row += 1;
        draw_text(
            buffer,
            x,
            row,
            &format!(
                "{} critical  {} high",
                count_severity(self.report, RiskLevel::Critical),
                count_severity(self.report, RiskLevel::High)
            ),
            Style::fg(self.palette.muted),
        );
        row += 2;
        for (idx, view) in VIEWS.iter().enumerate() {
            let active = idx == self.active_view;
            let color = if active {
                self.palette.cyan
            } else {
                self.palette.line
            };
            draw_text(buffer, x, row, "│", Style::fg(color));
            if active {
                buffer.fill_rect(
                    x + 1,
                    row,
                    area.w.saturating_sub(4),
                    1,
                    self.palette.sidebar,
                );
            }
            draw_text(
                buffer,
                x + 2,
                row,
                &format!("{} {view}", idx + 1),
                if active {
                    Style::fg(self.palette.white).with_bold()
                } else {
                    Style::fg(self.palette.muted)
                },
            );
            row += 2;
        }

        let posture_y = row + 1;
        if posture_y + 8 >= area.y + area.h {
            return;
        }
        draw_text(
            buffer,
            x,
            posture_y,
            "filters",
            Style::fg(self.palette.muted),
        );
        draw_text(
            buffer,
            x,
            posture_y + 1,
            &format!("severity  {}", severity_filter_label(self.severity_filter)),
            Style::fg(self.palette.white),
        );
        draw_text(
            buffer,
            x,
            posture_y + 2,
            &format!(
                "search    {}",
                truncate(&search_label(&self.search_query, self.search_mode), 12)
            ),
            Style::fg(self.palette.white),
        );
        draw_text(
            buffer,
            x,
            posture_y + 5,
            "keys",
            Style::fg(self.palette.muted),
        );
        draw_text(
            buffer,
            x,
            posture_y + 6,
            "tab/1-7 navigate",
            Style::fg(self.palette.muted),
        );
        draw_text(
            buffer,
            x,
            posture_y + 7,
            "/ search  s severity",
            Style::fg(self.palette.muted),
        );
        draw_text(
            buffer,
            x,
            posture_y + 8,
            "q quit",
            Style::fg(self.palette.muted),
        );
    }

    fn render_content(&self, buffer: &mut OptimizedBuffer, area: Area) {
        match self.active_view {
            0 => self.render_overview(buffer, area),
            1 => {
                let list_w = responsive_width(area.w, 48, 34, 36);
                self.render_findings(buffer, Area::new(area.x, area.y, list_w, area.h));
                self.render_detail(
                    buffer,
                    Area::new(
                        area.x + list_w + 2,
                        area.y,
                        area.w.saturating_sub(list_w + 2),
                        area.h,
                    ),
                );
            }
            2 => self.render_analysis(buffer, area),
            3 => self.render_fix_plan(buffer, area),
            4 => self.render_inventory(buffer, area),
            5 => self.render_backup(buffer, area),
            _ => self.render_help(buffer, area),
        }
    }

    fn render_overview(&self, buffer: &mut OptimizedBuffer, area: Area) {
        let left_w = responsive_width(area.w, 39, 30, 42);
        let right_x = area.x + left_w + 2;
        let right_w = area.w.saturating_sub(left_w + 2);
        self.render_risk_posture(
            buffer,
            Area::new(area.x, area.y + 1, left_w, area.h.saturating_sub(2)),
        );
        self.render_recent_findings(
            buffer,
            Area::new(right_x, area.y + 1, right_w, area.h.saturating_sub(2)),
        );
    }

    fn render_risk_posture(&self, buffer: &mut OptimizedBuffer, area: Area) {
        let risk = max_risk(&self.report.findings);
        let bottom = area.y + area.h.saturating_sub(1);
        plain_box(
            buffer,
            area,
            severity_color(&self.palette, risk),
            self.palette.panel,
        );
        let mut row = area.y + 2;
        draw_text(
            buffer,
            area.x + 2,
            row,
            "risk posture",
            Style::fg(self.palette.white).with_bold(),
        );
        row += 3;
        let total = self.report.summary.total_findings.max(1);
        for severity in [
            RiskLevel::Critical,
            RiskLevel::High,
            RiskLevel::Medium,
            RiskLevel::Low,
            RiskLevel::Info,
        ] {
            let count = count_severity(self.report, severity);
            let label = risk_word(severity);
            let bar_width = area.w.saturating_sub(18).clamp(4, 20) as usize;
            let bar = ascii_bar(count, total, bar_width);
            draw_text(
                buffer,
                area.x + 2,
                row,
                &format!("{label:<9} {bar} {count}"),
                Style::fg(severity_color(&self.palette, severity)),
            );
            row += 1;
        }
        row += 1;
        if row + 3 >= bottom {
            return;
        }
        draw_text(
            buffer,
            area.x + 2,
            row,
            "next action",
            Style::fg(self.palette.cyan).with_bold(),
        );
        row += 1;
        for line in wrap(&next_action(self.report), area.w.saturating_sub(4) as usize)
            .into_iter()
            .take(3)
        {
            draw_text(
                buffer,
                area.x + 2,
                row,
                &line,
                Style::fg(self.palette.white),
            );
            row += 1;
        }
        row += 1;
        if row + 4 >= bottom {
            return;
        }
        draw_text(
            buffer,
            area.x + 2,
            row,
            "safe defaults",
            Style::fg(self.palette.cyan).with_bold(),
        );
        row += 1;
        for line in [
            "read-only scan",
            "redacted outputs",
            "offline unless requested",
        ] {
            if row >= bottom {
                break;
            }
            draw_text(buffer, area.x + 2, row, line, Style::fg(self.palette.white));
            row += 1;
        }
    }

    fn render_recent_findings(&self, buffer: &mut OptimizedBuffer, area: Area) {
        plain_box(buffer, area, self.palette.blue, self.palette.surface);
        let mut row = area.y + 2;
        draw_text(
            buffer,
            area.x + 2,
            row,
            "recent findings",
            Style::fg(self.palette.white).with_bold(),
        );
        row += 3;
        let show_flow = area.h >= 18;
        let max_rows = if show_flow {
            (area.h.saturating_sub(13) as usize / 3).clamp(1, 5)
        } else {
            (area.h.saturating_sub(5) as usize / 3).clamp(1, 2)
        };
        for finding in self.display_findings().into_iter().take(max_rows) {
            let color = severity_color(&self.palette, finding.severity);
            draw_text(
                buffer,
                area.x + 3,
                row,
                severity_badge(finding.severity),
                Style::fg(color).with_bold(),
            );
            draw_text(
                buffer,
                area.x + 12,
                row,
                &truncate(&finding.rule, area.w.saturating_sub(16) as usize),
                Style::fg(color).with_bold(),
            );
            row += 1;
            draw_text(
                buffer,
                area.x + 3,
                row,
                &truncate(&recent_message(finding), area.w.saturating_sub(6) as usize),
                Style::fg(self.palette.muted),
            );
            row += 2;
        }
        if !show_flow {
            return;
        }
        let flow_y = area.y + area.h.saturating_sub(8);
        draw_text(
            buffer,
            area.x + 2,
            flow_y,
            "review flow",
            Style::fg(self.palette.cyan).with_bold(),
        );
        for (idx, line) in [
            "inspect finding evidence",
            "export plan-only fix material",
            "apply changes manually after review",
        ]
        .iter()
        .enumerate()
        {
            draw_text(
                buffer,
                area.x + 2,
                flow_y + 1 + u32::try_from(idx).unwrap_or(0),
                &format!("{}. {line}", idx + 1),
                Style::fg(self.palette.white),
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
            "SEVERITY",
            Style::fg(self.palette.muted),
        );
        draw_text(
            buffer,
            area.x + 12,
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
        let display_findings = self.display_findings();
        for (idx, finding) in display_findings.into_iter().take(max_rows).enumerate() {
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
                area.x + 12,
                row,
                &truncate(&finding.rule, area.w.saturating_sub(29) as usize),
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

        let visible_findings = self.display_findings();
        if let Some(finding) = visible_findings.get(self.selected_finding) {
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

    fn display_findings(&self) -> Vec<&Finding> {
        let mut findings = self.report.findings.iter().collect::<Vec<_>>();
        if let Some(severity) = self.severity_filter {
            findings.retain(|finding| finding.severity == severity);
        }
        let query = self.search_query.trim().to_ascii_lowercase();
        if !query.is_empty() {
            findings.retain(|finding| {
                finding.rule.to_ascii_lowercase().contains(&query)
                    || finding.server.to_ascii_lowercase().contains(&query)
                    || finding.path.to_ascii_lowercase().contains(&query)
                    || finding.message.to_ascii_lowercase().contains(&query)
                    || finding.id.to_ascii_lowercase().contains(&query)
            });
        }
        findings.sort_by(|a, b| {
            b.severity
                .rank()
                .cmp(&a.severity.rank())
                .then_with(|| rule_display_rank(&a.rule).cmp(&rule_display_rank(&b.rule)))
                .then_with(|| a.server.cmp(&b.server))
                .then_with(|| a.id.cmp(&b.id))
        });
        findings
    }

    fn render_analysis(&self, buffer: &mut OptimizedBuffer, area: Area) {
        let report = analysis::run(
            self.report,
            AnalysisOptions {
                mode: self.report.scan_mode.clone(),
                workspace: self.report.workspace.clone(),
                with: Vec::new(),
                online: false,
                package: String::new(),
                finding_id: String::new(),
            },
        );
        let left_w = responsive_width(area.w, 42, 32, 42);
        let right_x = area.x + left_w + 2;
        let right_w = area.w.saturating_sub(left_w + 2);

        draw_panel(
            buffer,
            Area::new(area.x, area.y, left_w, area.h),
            "Analysis Summary",
            self.palette.magenta,
            self.palette.panel,
        );
        let mut row = area.y + 2;
        for (label, value, color) in [
            (
                "signals",
                report.summary.total_signals.to_string(),
                self.palette.magenta,
            ),
            (
                "subjects",
                report.summary.total_subjects.to_string(),
                self.palette.blue,
            ),
            (
                "provider warnings",
                report.summary.provider_warnings.to_string(),
                self.palette.amber,
            ),
            (
                "highest",
                risk_word(report.summary.highest_severity).to_string(),
                severity_color(&self.palette, report.summary.highest_severity),
            ),
        ] {
            draw_text(
                buffer,
                area.x + 2,
                row,
                label,
                Style::fg(self.palette.muted),
            );
            draw_text(
                buffer,
                area.x + left_w.saturating_sub(18),
                row,
                &truncate(&value, 16),
                Style::fg(color).with_bold(),
            );
            row += 2;
        }
        row += 1;
        section_label(buffer, area.x + 2, row, "categories", self.palette.cyan);
        row += 2;
        for (category, count) in report
            .summary
            .signals_by_category
            .iter()
            .take(area.h.saturating_sub(15) as usize)
        {
            draw_text(
                buffer,
                area.x + 2,
                row,
                &truncate(&format!("{category:?}").to_ascii_lowercase(), 20),
                Style::fg(self.palette.white),
            );
            draw_text(
                buffer,
                area.x + left_w.saturating_sub(6),
                row,
                &count.to_string(),
                Style::fg(self.palette.cyan).with_bold(),
            );
            row += 1;
        }

        draw_panel(
            buffer,
            Area::new(right_x, area.y, right_w, area.h),
            "Signals",
            self.palette.blue,
            self.palette.surface,
        );
        let mut row = area.y + 2;
        for signal in report
            .signals
            .iter()
            .take(area.h.saturating_sub(5) as usize / 4)
        {
            let color = severity_color(&self.palette, signal.severity);
            draw_text(
                buffer,
                right_x + 2,
                row,
                severity_badge(signal.severity),
                Style::fg(color).with_bold(),
            );
            draw_text(
                buffer,
                right_x + 12,
                row,
                &truncate(&signal.rule, right_w.saturating_sub(16) as usize),
                Style::fg(color).with_bold(),
            );
            row += 1;
            for line in wrap(&signal.message, right_w.saturating_sub(4) as usize)
                .into_iter()
                .take(2)
            {
                draw_text(
                    buffer,
                    right_x + 2,
                    row,
                    &line,
                    Style::fg(self.palette.muted),
                );
                row += 1;
            }
            row += 1;
        }
    }

    fn render_fix_plan(&self, buffer: &mut OptimizedBuffer, area: Area) {
        let plan = fixplan::plan(
            self.report,
            Selector {
                all: true,
                ..Selector::default()
            },
        );
        let left_w = responsive_width(area.w, 36, 30, 42);
        let right_x = area.x + left_w + 2;
        let right_w = area.w.saturating_sub(left_w + 2);

        draw_panel(
            buffer,
            Area::new(area.x, area.y, left_w, area.h),
            "Plan Summary",
            self.palette.amber,
            self.palette.panel,
        );
        let mut row = area.y + 2;
        for (label, value, color) in [
            ("safe", plan.summary.safe, self.palette.green),
            ("review", plan.summary.review, self.palette.amber),
            ("blocked", plan.summary.blocked, self.palette.red),
        ] {
            draw_text(
                buffer,
                area.x + 2,
                row,
                label,
                Style::fg(self.palette.muted),
            );
            draw_text(
                buffer,
                area.x + left_w.saturating_sub(6),
                row,
                &value.to_string(),
                Style::fg(color).with_bold(),
            );
            row += 2;
        }
        row += 1;
        section_label(buffer, area.x + 2, row, "groups", self.palette.cyan);
        row += 2;
        for group in plan
            .groups
            .iter()
            .take(area.h.saturating_sub(13) as usize / 2)
        {
            draw_text(
                buffer,
                area.x + 2,
                row,
                &truncate(&group.title, left_w.saturating_sub(12) as usize),
                Style::fg(severity_color(&self.palette, group.severity)).with_bold(),
            );
            draw_text(
                buffer,
                area.x + left_w.saturating_sub(6),
                row,
                &group.finding_count.to_string(),
                Style::fg(self.palette.white),
            );
            row += 2;
        }

        draw_panel(
            buffer,
            Area::new(right_x, area.y, right_w, area.h),
            "Plan-Only Actions",
            self.palette.amber,
            self.palette.surface,
        );
        let mut row = area.y + 2;
        for action in plan
            .actions
            .iter()
            .take(area.h.saturating_sub(5) as usize / 6)
        {
            let color = severity_color(&self.palette, action.severity);
            draw_text(
                buffer,
                right_x + 2,
                row,
                &action.title,
                Style::fg(color).with_bold(),
            );
            draw_text(
                buffer,
                right_x + right_w.saturating_sub(12),
                row,
                risk_word(action.severity),
                Style::fg(color),
            );
            row += 1;
            draw_text(
                buffer,
                right_x + 2,
                row,
                &truncate(&action.finding_id, right_w.saturating_sub(4) as usize),
                Style::fg(self.palette.muted),
            );
            row += 1;
            for line in action.steps.iter().take(2) {
                draw_text(
                    buffer,
                    right_x + 2,
                    row,
                    &truncate(line, right_w.saturating_sub(4) as usize),
                    Style::fg(self.palette.white),
                );
                row += 1;
            }
            row += 1;
        }
    }

    fn render_inventory(&self, buffer: &mut OptimizedBuffer, area: Area) {
        let left_w = responsive_width(area.w, 38, 30, 42);
        let right_x = area.x + left_w + 2;
        let right_w = area.w.saturating_sub(left_w + 2);

        draw_panel(
            buffer,
            Area::new(area.x, area.y, left_w, area.h),
            "Inventory",
            self.palette.blue,
            self.palette.panel,
        );
        let mut row = area.y + 2;
        for (classification, count) in self
            .report
            .summary
            .items_by_classification
            .iter()
            .take(area.h.saturating_sub(5) as usize)
        {
            let color = classification_color(&self.palette, *classification);
            draw_text(
                buffer,
                area.x + 2,
                row,
                &truncate(&format!("{classification:?}").to_ascii_lowercase(), 22),
                Style::fg(color).with_bold(),
            );
            draw_text(
                buffer,
                area.x + left_w.saturating_sub(6),
                row,
                &count.to_string(),
                Style::fg(self.palette.white),
            );
            row += 2;
        }

        draw_panel(
            buffer,
            Area::new(right_x, area.y, right_w, area.h),
            "Items",
            self.palette.blue,
            self.palette.surface,
        );
        let mut row = area.y + 2;
        for item in self
            .report
            .items
            .iter()
            .take(area.h.saturating_sub(4) as usize / 3)
        {
            let color = severity_color(&self.palette, item.risk);
            draw_text(
                buffer,
                right_x + 2,
                row,
                &truncate(&item.tool, 12),
                Style::fg(color).with_bold(),
            );
            draw_text(
                buffer,
                right_x + 16,
                row,
                &truncate(&item.kind, 18),
                Style::fg(self.palette.white),
            );
            row += 1;
            draw_text(
                buffer,
                right_x + 2,
                row,
                &truncate(&item.path, right_w.saturating_sub(4) as usize),
                Style::fg(self.palette.muted),
            );
            row += 2;
        }
    }

    fn render_backup(&self, buffer: &mut OptimizedBuffer, area: Area) {
        let plan = backupplan::plan(&self.report.home);
        let left_w = responsive_width(area.w, 45, 32, 40);
        let right_x = area.x + left_w + 2;
        let right_w = area.w.saturating_sub(left_w + 2);

        draw_panel(
            buffer,
            Area::new(area.x, area.y, left_w, area.h),
            "Portable Candidates",
            self.palette.green,
            self.palette.panel,
        );
        let mut row = area.y + 2;
        draw_text(
            buffer,
            area.x + 2,
            row,
            "include",
            Style::fg(self.palette.cyan).with_bold(),
        );
        row += 2;
        for item in plan
            .include
            .iter()
            .take(area.h.saturating_sub(8) as usize / 2)
        {
            draw_text(buffer, area.x + 2, row, "＋", Style::fg(self.palette.green));
            draw_text(
                buffer,
                area.x + 5,
                row,
                &truncate(item, left_w.saturating_sub(7) as usize),
                Style::fg(self.palette.white),
            );
            row += 2;
        }

        draw_panel(
            buffer,
            Area::new(right_x, area.y, right_w, area.h),
            "Never Sync",
            self.palette.red,
            self.palette.surface,
        );
        let mut row = area.y + 2;
        draw_text(
            buffer,
            right_x + 2,
            row,
            "exclude",
            Style::fg(self.palette.red).with_bold(),
        );
        row += 2;
        for item in plan
            .exclude
            .iter()
            .take(area.h.saturating_sub(11) as usize / 2)
        {
            draw_text(buffer, right_x + 2, row, "−", Style::fg(self.palette.red));
            draw_text(
                buffer,
                right_x + 5,
                row,
                &truncate(item, right_w.saturating_sub(7) as usize),
                Style::fg(self.palette.white),
            );
            row += 2;
        }
        row += 1;
        section_label(buffer, right_x + 2, row, "notes", self.palette.cyan);
        row += 2;
        for note in plan.notes.iter().take(3) {
            for line in wrap(note, right_w.saturating_sub(4) as usize)
                .into_iter()
                .take(2)
            {
                draw_text(
                    buffer,
                    right_x + 2,
                    row,
                    &line,
                    Style::fg(self.palette.muted),
                );
                row += 1;
            }
        }
    }

    fn render_help(&self, buffer: &mut OptimizedBuffer, area: Area) {
        let left_w = responsive_width(area.w, 45, 32, 40);
        let right_x = area.x + left_w + 2;
        let right_w = area.w.saturating_sub(left_w + 2);

        draw_panel(
            buffer,
            Area::new(area.x, area.y, left_w, area.h),
            "Keyboard",
            self.palette.cyan,
            self.palette.panel,
        );
        let mut row = area.y + 2;
        for (key, label) in [
            ("tab / shift-tab", "move between sections"),
            ("1-7", "jump to a section"),
            ("j / down", "next finding"),
            ("k / up", "previous finding"),
            ("q / esc", "quit"),
        ] {
            draw_text(
                buffer,
                area.x + 2,
                row,
                key,
                Style::fg(self.palette.cyan).with_bold(),
            );
            draw_text(
                buffer,
                area.x + 20,
                row,
                label,
                Style::fg(self.palette.white),
            );
            row += 2;
        }

        draw_panel(
            buffer,
            Area::new(right_x, area.y, right_w, area.h),
            "Safety Model",
            self.palette.green,
            self.palette.surface,
        );
        let mut row = area.y + 2;
        for line in [
            "No live config mutation from the TUI.",
            "Fixes are plan-only review material.",
            "Evidence is redacted before display.",
            "Online providers require explicit CLI flags.",
            "Use report diff/history for follow-up review.",
        ] {
            draw_text(buffer, right_x + 2, row, "•", Style::fg(self.palette.green));
            for wrapped in wrap(line, right_w.saturating_sub(7) as usize)
                .into_iter()
                .take(2)
            {
                draw_text(
                    buffer,
                    right_x + 5,
                    row,
                    &wrapped,
                    Style::fg(self.palette.white),
                );
                row += 1;
            }
            row += 1;
        }
    }
}

#[derive(Clone, Copy)]
struct Palette {
    bg: Rgba,
    sidebar: Rgba,
    panel: Rgba,
    surface: Rgba,
    code_bg: Rgba,
    line: Rgba,
    white: Rgba,
    muted: Rgba,
    cyan: Rgba,
    green: Rgba,
    lime: Rgba,
    amber: Rgba,
    red: Rgba,
    orange: Rgba,
    magenta: Rgba,
    blue: Rgba,
}

impl Palette {
    fn new() -> Self {
        Self {
            bg: color("#080808"),
            sidebar: color("#1C1C1C"),
            panel: color("#1C1C1C"),
            surface: color("#1C1C1C"),
            code_bg: color("#111827"),
            line: color("#26324A"),
            white: color("#E8EEF8"),
            muted: color("#7D8799"),
            cyan: color("#2DD4BF"),
            green: color("#34D399"),
            lime: color("#A3E635"),
            amber: color("#FFD166"),
            red: color("#FF4D6D"),
            orange: color("#FF8A3D"),
            magenta: color("#A78BFA"),
            blue: color("#60A5FA"),
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
    options.fill = Some(fill);
    options.title = Some(format!(" {title} "));
    options.title_align = TitleAlign::Left;
    buffer.draw_box_with_options(area.x, area.y, area.w, area.h, options);
}

fn plain_box(buffer: &mut OptimizedBuffer, area: Area, border: Rgba, fill: Rgba) {
    if area.w < 4 || area.h < 3 {
        return;
    }
    let mut options = BoxOptions::new(BoxStyle::rounded(Style::fg(border)));
    options.fill = Some(fill);
    buffer.draw_box_with_options(area.x, area.y, area.w, area.h, options);
}

fn old_stat_card(
    buffer: &mut OptimizedBuffer,
    area: Area,
    label: &str,
    value: &str,
    color: Rgba,
    palette: &Palette,
) {
    plain_box(buffer, area, color, palette.panel);
    draw_text(
        buffer,
        area.x + 2,
        area.y + 1,
        label,
        Style::fg(palette.muted),
    );
    draw_text(
        buffer,
        area.x + 2,
        area.y + 2,
        &truncate(value, area.w.saturating_sub(4) as usize),
        Style::fg(color).with_bold(),
    );
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

fn draw_hline(buffer: &mut OptimizedBuffer, x: u32, y: u32, width: u32, color: Rgba) {
    for offset in 0..width {
        draw_text(buffer, x + offset, y, "─", Style::fg(color));
    }
}

fn draw_vline(buffer: &mut OptimizedBuffer, x: u32, y: u32, height: u32, color: Rgba) {
    for offset in 0..height {
        draw_text(buffer, x, y + offset, "│", Style::fg(color));
    }
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

fn classification_color(palette: &Palette, classification: Classification) -> Rgba {
    match classification {
        Classification::Portable => palette.green,
        Classification::MachineLocal => palette.amber,
        Classification::SecretAuth => palette.red,
        Classification::RuntimeCache | Classification::AppOwned => palette.magenta,
        Classification::Unknown => palette.muted,
    }
}

fn view_nav_color(palette: &Palette, idx: usize) -> Rgba {
    match idx {
        0 => palette.cyan,
        1 => palette.red,
        2 => palette.magenta,
        3 => palette.amber,
        4 => palette.blue,
        5 => palette.green,
        _ => palette.white,
    }
}

fn severity_color(palette: &Palette, risk: RiskLevel) -> Rgba {
    match risk {
        RiskLevel::Critical => palette.red,
        RiskLevel::High => palette.orange,
        RiskLevel::Medium => palette.amber,
        RiskLevel::Low => palette.blue,
        RiskLevel::Info => palette.lime,
    }
}

fn rule_display_rank(rule: &str) -> usize {
    match rule {
        "mcp_unpinned_package" => 0,
        "mcp_secret_env" | "mcp_secret_header" => 1,
        "mcp_broad_filesystem" => 2,
        "mcp_server_review" => 3,
        _ => 10,
    }
}

fn recent_message(finding: &Finding) -> String {
    match finding.rule.as_str() {
        "mcp_unpinned_package" => {
            format!("MCP server \"{}\" runs a package executor", finding.server)
        }
        "mcp_secret_env" | "mcp_secret_header" => {
            format!("MCP server \"{}\" references a sensitive", finding.server)
        }
        "mcp_broad_filesystem" => {
            format!(
                "MCP server \"{}\" appears to reference broad",
                finding.server
            )
        }
        "mcp_server_review" => {
            format!(
                "Review MCP server \"{}\" before syncing this",
                finding.server
            )
        }
        _ => finding.message.clone(),
    }
}

fn severity_badge(risk: RiskLevel) -> &'static str {
    match risk {
        RiskLevel::Critical => "CRITICAL",
        RiskLevel::High => "HIGH",
        RiskLevel::Medium => "MEDIUM",
        RiskLevel::Low => "LOW",
        RiskLevel::Info => "INFO",
    }
}

fn risk_label(risk: RiskLevel) -> &'static str {
    match risk {
        RiskLevel::Critical => "CRITICAL",
        RiskLevel::High => "HIGH",
        RiskLevel::Medium => "MEDIUM",
        RiskLevel::Low => "LOW",
        RiskLevel::Info => "INFO",
    }
}

fn risk_word(risk: RiskLevel) -> &'static str {
    match risk {
        RiskLevel::Critical => "critical",
        RiskLevel::High => "high",
        RiskLevel::Medium => "medium",
        RiskLevel::Low => "low",
        RiskLevel::Info => "info",
    }
}

fn ascii_bar(value: usize, total: usize, width: usize) -> String {
    let filled = if total == 0 {
        0
    } else {
        value.saturating_mul(width) / total
    };
    format!(
        "{}{}",
        "#".repeat(filled),
        "-".repeat(width.saturating_sub(filled))
    )
}

fn next_action(report: &Report) -> String {
    match max_risk(&report.findings) {
        RiskLevel::Critical => {
            "Externalize inline secrets before syncing these configs.".to_string()
        }
        RiskLevel::High => "Pin package executors and review remote MCP wrappers.".to_string(),
        RiskLevel::Medium => "Review local endpoints and machine-specific paths.".to_string(),
        _ => "Keep this report as the clean baseline for future diffs.".to_string(),
    }
}

fn status_label(risk: RiskLevel, total: usize) -> String {
    match risk {
        RiskLevel::Critical => format!("{total} findings / critical"),
        RiskLevel::High => format!("{total} findings / high"),
        RiskLevel::Medium => format!("{total} findings / medium"),
        _ => "OK".to_string(),
    }
}

fn severity_filter_label(filter: Option<RiskLevel>) -> &'static str {
    match filter {
        Some(risk) => risk_word(risk),
        None => "all",
    }
}

fn search_label(query: &str, active: bool) -> String {
    if active {
        format!("/{query}")
    } else if query.is_empty() {
        "none".to_string()
    } else {
        query.to_string()
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

fn responsive_width(total: u32, percent: u32, minimum: u32, reserve: u32) -> u32 {
    let maximum = total.saturating_sub(reserve);
    if maximum <= minimum {
        return maximum.max(1);
    }
    (total.saturating_mul(percent) / 100).clamp(minimum, maximum)
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
    use nightward_core::inventory::load_report;
    use opentui::{CellContent, KeyEvent};
    use std::path::PathBuf;

    fn fixture_report() -> Report {
        let path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("../../site/public/demo/nightward-sample-scan.json");
        load_report(path).expect("fixture scan report")
    }

    fn render_text(app: &TuiState<'_>, width: u32, height: u32) -> String {
        let mut buffer = OptimizedBuffer::new(width, height);
        app.render(&mut buffer, width, height);
        let mut out = String::new();
        for y in 0..height {
            for x in 0..width {
                let ch = buffer
                    .get(x, y)
                    .and_then(|cell| match cell.content {
                        CellContent::Char(ch) => Some(ch),
                        CellContent::Empty | CellContent::Continuation => Some(' '),
                        CellContent::Grapheme(_) => None,
                    })
                    .unwrap_or(' ');
                out.push(ch);
            }
            out.push('\n');
        }
        out
    }

    fn cell_char(buffer: &OptimizedBuffer, x: u32, y: u32) -> char {
        buffer
            .get(x, y)
            .and_then(|cell| match cell.content {
                CellContent::Char(ch) => Some(ch),
                CellContent::Empty | CellContent::Continuation => Some(' '),
                CellContent::Grapheme(_) => None,
            })
            .unwrap_or(' ')
    }

    #[test]
    fn severity_labels_are_short() {
        assert_eq!(severity_badge(RiskLevel::Critical), "CRITICAL");
        assert_eq!(severity_badge(RiskLevel::Info), "INFO");
    }

    #[test]
    fn wraps_long_text_without_dropping_words() {
        let lines = wrap("one two three four", 8);
        assert_eq!(lines, vec!["one two", "three", "four"]);
    }

    #[test]
    fn renders_all_views_with_real_content_at_common_sizes() {
        let report = fixture_report();
        for (width, height) in [(80, 24), (120, 36), (144, 45)] {
            for (view, view_name) in VIEWS.iter().enumerate() {
                let mut app = TuiState::new(&report);
                app.active_view = view;
                let text = render_text(&app, width, height);
                assert!(
                    text.contains(view_name),
                    "view {} should render its title at {width}x{height}",
                    view_name
                );
                assert!(
                    !text
                        .lines()
                        .skip((height / 2) as usize)
                        .collect::<String>()
                        .trim()
                        .is_empty(),
                    "lower half should not be blank for {} at {width}x{height}",
                    view_name
                );
            }
        }
    }

    #[test]
    fn keyboard_navigation_switches_views_and_findings() {
        let report = fixture_report();
        let mut app = TuiState::new(&report);

        assert!(app.handle_event(&Event::Key(KeyEvent::char('2'))));
        assert_eq!(app.active_view, 1);
        let first = app.selected_finding;
        assert!(app.handle_event(&Event::Key(KeyEvent::key(KeyCode::Down))));
        assert_ne!(app.selected_finding, first);
        assert!(app.handle_event(&Event::Key(KeyEvent::key(KeyCode::Up))));
        assert_eq!(app.selected_finding, first);
        assert!(app.handle_event(&Event::Key(KeyEvent::key(KeyCode::Tab))));
        assert_eq!(app.active_view, 2);
        assert!(app.handle_event(&Event::Key(KeyEvent::key(KeyCode::Left))));
        assert_eq!(app.active_view, 1);
        assert!(app.handle_event(&Event::Key(KeyEvent::key(KeyCode::Right))));
        assert_eq!(app.active_view, 2);
        assert!(!app.handle_event(&Event::Key(KeyEvent::char('q'))));
    }

    #[test]
    fn keyboard_filters_findings_and_keeps_detail_aligned() {
        let report = fixture_report();
        let mut app = TuiState::new(&report);
        app.active_view = 1;

        assert_eq!(app.display_findings().len(), report.findings.len());
        assert!(app.handle_event(&Event::Key(KeyEvent::char('s'))));
        assert_eq!(app.severity_filter, Some(RiskLevel::Critical));
        assert!(app.display_findings().len() <= report.findings.len());
        assert!(app.handle_event(&Event::Key(KeyEvent::char('s'))));
        assert_eq!(app.severity_filter, Some(RiskLevel::High));
        assert_eq!(app.display_findings().len(), 1);

        assert!(app.handle_event(&Event::Key(KeyEvent::char('/'))));
        for ch in "package".chars() {
            assert!(app.handle_event(&Event::Key(KeyEvent::char(ch))));
        }
        assert!(app.handle_event(&Event::Key(KeyEvent::key(KeyCode::Enter))));
        let filtered = app.display_findings();
        assert_eq!(filtered.len(), 1);
        assert_eq!(filtered[0].rule, "mcp_unpinned_package");

        let text = render_text(&app, 120, 36);
        assert!(text.contains("mcp_unpinned_package"));
        assert!(!text.contains("mcp_secret_env"));

        assert!(app.handle_event(&Event::Key(KeyEvent::char('x'))));
        assert_eq!(app.severity_filter, None);
        assert!(app.search_query.is_empty());
        assert_eq!(app.display_findings().len(), report.findings.len());
    }

    #[test]
    fn compact_risk_bars_do_not_overwrite_panel_borders() {
        let report = fixture_report();
        let app = TuiState::new(&report);
        let mut buffer = OptimizedBuffer::new(80, 24);
        app.render(&mut buffer, 80, 24);

        for y in 12..=16 {
            assert_eq!(cell_char(&buffer, 30, y), '│');
        }
    }
}
