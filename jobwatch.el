;;; jobwatch.el --- table of Loom jobs, evidence, and a re-spacing plan -*- lexical-binding: t -*-

;; Copyright (C) 2026 Curtis
;;
;; Licensed under the Apache License, Version 2.0.

;;; Commentary:

;; jobwatch reads `mint-jw report --json` and lists each job. A row opens
;; the evidence for that job. Where Emacs can draw, the recent successful
;; run lengths are an SVG. `jobwatch-respace` shows the plan from
;; `mint-jw respace` and does not apply it.
;;
;; Needs Emacs 27.

;;; Code:

(require 'tabulated-list)
(require 'seq)
(require 'subr-x)
(require 'xml)

(defgroup jobwatch nil
  "Loom jobs in Emacs."
  :group 'external)

(defcustom jobwatch-program "mint-jw"
  "Program that prints the job report."
  :type 'string
  :group 'jobwatch)

(defcustom jobwatch-window "2h"
  "How far `jobwatch-respace' may move a start, passed to mint-jw."
  :type 'string
  :group 'jobwatch)

(defvar jobwatch--report nil
  "Last parsed report, an alist.")

(defun jobwatch--call (args)
  "Run `jobwatch-program' with ARGS. Return (exit-code . output)."
  (let ((buf (generate-new-buffer " *jobwatch*")))
    (unwind-protect
        (cons (apply #'call-process jobwatch-program nil buf nil args)
              (with-current-buffer buf (buffer-string)))
      (kill-buffer buf))))

(defun jobwatch--parse (text)
  "Parse TEXT from mint-jw --json into an alist."
  (json-parse-string text :object-type 'alist :array-type 'list :null-object nil))

(defun jobwatch--get (obj key)
  "Field KEY of alist OBJ. KEY is a string or a symbol.
JSON false is the symbol :false."
  (alist-get (if (symbolp key) key (intern key)) obj))

(defun jobwatch--field (obj key)
  "String or number field KEY of alist OBJ, as a string."
  (let ((v (jobwatch--get obj key)))
    (cond
     ((or (null v) (eq v :false)) "")
     ((stringp v) v)
     (t (format "%s" v)))))

(defun jobwatch--ingest (text)
  "Store the report in TEXT and return its jobs."
  (setq jobwatch--report (jobwatch--parse text))
  (jobwatch--get jobwatch--report "jobs"))

(defun jobwatch--rows (jobs)
  "Tabulated-list entries for JOBS."
  (mapcar
   (lambda (job)
     (cons (jobwatch--field job "name")
           (vector (jobwatch--field job "name")
                   (jobwatch--field job "state")
                   (jobwatch--field job "level")
                   (jobwatch--field job "runs")
                   (jobwatch--field job "says"))))
   jobs))

(defun jobwatch--findings (name)
  "Findings in the last report whose job is NAME."
  (seq-filter
   (lambda (f) (equal (jobwatch--get f "job") name))
   (jobwatch--get jobwatch--report "findings")))

(defun jobwatch--evidence-text (name)
  "Evidence buffer text for the job NAME."
  (let ((fs (jobwatch--findings name))
        (out (format "Evidence for %s\n\n" name)))
    (if (null fs)
        (concat out "No finding for this job.\n")
      (dolist (f fs)
        (setq out (concat out (jobwatch--field f "level") "  "
                          (jobwatch--field f "kind") "\n"
                          (jobwatch--field f "says") "\n"))
        (dolist (line (jobwatch--get f "evidence"))
          (setq out (concat out "  " (format "%s" line) "\n")))
        (setq out (concat out "\n")))
      out)))

(defun jobwatch--svg (name durations)
  "SVG of DURATIONS, in seconds, for the job NAME. Empty when there are none."
  (when durations
    (let* ((vals (mapcar (lambda (n) (float n)) durations))
           (hi (apply #'max vals))
           (lo (apply #'min vals))
           (span (max 1.0 (- hi lo)))
           (w 480)
           (h 140)
           (n (length vals))
           (pts "")
           (i 0))
      (dolist (v vals)
        (let ((x (if (< n 2) (/ w 2) (round (* (/ (float i) (1- n)) (- w 40)))))
              (y (round (- h 24 (* (/ (- v lo) span) (- h 48))))))
          (setq pts (concat pts (format "%d,%d " (+ 20 x) y)))
          (setq i (1+ i))))
      (format "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"%d\" height=\"%d\"><text x=\"8\" y=\"16\">%s, seconds</text><polyline fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" points=\"%s\"/></svg>"
              w h (xml-escape-string name) (string-trim pts)))))

(defun jobwatch--plan-text (plan)
  "Text of a re-spacing PLAN. It says the plan was not applied."
  (let ((out "Re-spacing plan\n\nNothing was changed. Apply a move by editing that job's schedule yourself.\n\n"))
    (setq out (concat out "Before: " (jobwatch--field plan "before") "\n"
                      "After: " (jobwatch--field plan "after") "\n"
                      (jobwatch--field plan "effect") "\n\n"))
    (dolist (m (jobwatch--get plan "moves"))
      (setq out (concat out (jobwatch--field m "job") "  "
                        (jobwatch--field m "from") " -> "
                        (jobwatch--field m "to") "\n")))
    (let ((applied (jobwatch--get plan "applied")))
      (when (and applied (not (eq applied :false)))
        (setq out (concat out "\nThe report said a move was applied. This buffer did not apply one.\n"))))
    out))

;;; The day

(defvar jobwatch--timeline nil
  "Last parsed timeline, an alist.")

(defvar jobwatch--timeline-row nil
  "The job the cursor is on in the timeline, by name.")

(defun jobwatch--minutes (m)
  "M as a clock time, \"02:30\". Past midnight wraps to the next day."
  (let* ((n (round m))
         (d (floor n 1440))
         (r (- n (* d 1440)))
         (s (format "%02d:%02d" (floor r 60) (mod r 60))))
    (if (> d 0) (format "%s+%dd" s d) s)))

(defun jobwatch--timeline-bars ()
  "The bars of the last timeline."
  (or (jobwatch--get jobwatch--timeline "bars") '()))

(defun jobwatch--timeline-text ()
  "The timeline as text: an axis, then one line per job.
Each line is the job, its usual run, the gap before the next job, and its
state. A job whose slowest runs already reach the next start says so."
  (let ((bars (jobwatch--timeline-bars))
        (clock (jobwatch--field jobwatch--timeline "clock"))
        (width 16)
        (out ""))
    (if (null bars)
        (concat "No job has enough runs to draw yet.\n"
                "A job appears once it has three recent successful runs by its schedule.\n")
      ;; The name column is as wide as the longest name, so the clock
      ;; times line up under each other.
      (dolist (b bars)
        (setq width (max width (length (jobwatch--field b "name")))))
      (setq out (concat (format (format "%%-%ds %%-7s %%-7s %%-7s %%-7s %%s\n" width)
                                "Job" "Starts" "Usual" "Slowest" "Next" "Gap and state")
                        clock "\n"))
      (dolist (b bars)
        (let* ((name (jobwatch--field b "name"))
               (next (jobwatch--get b "next"))
               ;; JSON false is the symbol :false, which is not nil, so
               ;; it has to be turned into a real false.
               (over (let ((v (jobwatch--get b "overrun")))
                       (and v (not (eq v :false)))))
               (gap (jobwatch--get b "gap"))
               (next-name (jobwatch--field b "nextName"))
               (low (let ((v (jobwatch--get b "lowConfidence")))
                      (and v (not (eq v :false)))))
               (says (jobwatch--field b "says"))
               (state (jobwatch--field b "state"))
               (where (cond
                       ((or (null next) (eq next :false) (not (numberp next)) (< (float next) 0))
                        "no job starts within 12 hours")
                       (over (if (string-empty-p next-name)
                         "its slowest runs reach midnight"
                       (format "runs into %s" next-name)))
                       ((and (numberp gap) (> (float gap) 0))
                        (format "%s before %s" (jobwatch--span gap) next-name))
                       (t (format "before %s" next-name)))))
          (setq out (concat out
                            (format (format "%%-%ds %%-7s %%-7s %%-7s %%-7s %%s%%s\n" width)
                                    name
                                    (jobwatch--minutes (float (jobwatch--get b "start")))
                                    (jobwatch--minutes (float (jobwatch--get b "usual")))
                                    (jobwatch--minutes (float (jobwatch--get b "slow")))
                                    (if (and (numberp next) (>= (float next) 0))
                                        (jobwatch--minutes (float next))
                                      "-")
                                    where
                                    (cond ((and over (eq (jobwatch--get b "period") 10080)) " (weekly)")
                                          (t "")))
                            (if low "                 (lower confidence)\n" "")
                            (if (string-empty-p says) ""
                              (format "                 %s: %s\n" state says))))))
      out)))

(defun jobwatch--span (minutes)
  "MINUTES said the way people do: 4 min, 1.5 hours, 2 days."
  (let ((m (float minutes)))
    (cond ((< m 1) "under a minute")
          ((< m 60) (format "%d min" (round m)))
          ((< m 2880) (format "%.1f hours" (/ m 60)))
          (t (format "%.0f days" (/ m 1440))))))

(defun jobwatch-timeline-text ()
  "The timeline as text, from the last `jobwatch-timeline'."
  (interactive)
  (let ((text (jobwatch--timeline-text)))
    (with-current-buffer (get-buffer-create "*jobwatch timeline*")
      (let ((inhibit-read-only t))
        (erase-buffer)
        (insert text)
        (goto-char (point-min))
        (special-mode))
      (display-buffer (current-buffer)))))

;;;###autoload
(defun jobwatch-timeline ()
  "Show one day of the jobs from `mint-jw timeline --json'.
Each job's usual run, its slowest runs, and the gap before the next job
starts. RET on a job opens its evidence, from the report."
  (interactive)
  (let* ((got (jobwatch--call '("timeline" "--json")))
         (code (car got)))
    (unless (or (= code 0) (= code 1))
      (error "mint-jw timeline failed: %s" (cdr got)))
    (setq jobwatch--timeline (jobwatch--parse (cdr got)))
    ;; The report carries the findings the evidence buffer shows. Reading it
    ;; too means a job opened from the timeline has the same evidence as the
    ;; same job opened from the list.
    (let ((rep (jobwatch--call '("report" "--json"))))
      (when (or (= (car rep) 0) (= (car rep) 1))
        (ignore-errors (jobwatch--ingest (cdr rep)))))
    (jobwatch-timeline-text)))

(define-derived-mode jobwatch-mode tabulated-list-mode "Jobwatch"
  "Table of scheduled jobs."
  (setq tabulated-list-format [("Job" 16 t) ("State" 12 t) ("Level" 10 t) ("Runs" 6 t) ("Says" 40 t)])
  (setq tabulated-list-padding 1)
  (tabulated-list-init-header))

(defun jobwatch--fill (jobs)
  "Show JOBS in the current buffer."
  (setq tabulated-list-entries (jobwatch--rows jobs))
  (tabulated-list-print t))

;;;###autoload
(defun jobwatch ()
  "List scheduled jobs from `mint-jw report --json'."
  (interactive)
  (let* ((got (jobwatch--call '("report" "--json")))
         (code (car got)))
    (unless (or (= code 0) (= code 1))
      (error "mint-jw report failed: %s" (cdr got)))
    (let ((jobs (jobwatch--ingest (cdr got))))
      (with-current-buffer (get-buffer-create "*jobwatch*")
        (jobwatch-mode)
        (jobwatch--fill jobs)
        (switch-to-buffer (current-buffer))))))

(defun jobwatch-evidence ()
  "Show the evidence for the job at point."
  (interactive)
  (let ((name (tabulated-list-get-id)))
    (unless name (user-error "No job on this line"))
    (with-current-buffer (get-buffer-create "*jobwatch evidence*")
      (let ((inhibit-read-only t))
        (erase-buffer)
        (insert (jobwatch--evidence-text name))
        (goto-char (point-min))
        (special-mode))
      (display-buffer (current-buffer)))))

(defun jobwatch-plot ()
  "Plot recent successful run lengths for the job at point.
Draws an SVG where Emacs can show images, and the SVG source otherwise."
  (interactive)
  (let* ((name (tabulated-list-get-id))
         (job (seq-find (lambda (j) (equal (jobwatch--get j "name") name))
                        (jobwatch--get jobwatch--report "jobs")))
         (durations (and job (jobwatch--get job "durations")))
         (svg (and durations (jobwatch--svg name durations))))
    (unless name (user-error "No job on this line"))
    (with-current-buffer (get-buffer-create "*jobwatch plot*")
      (let ((inhibit-read-only t))
        (erase-buffer)
        (cond
         ((null svg)
          (insert (format "No successful run lengths for %s.\n" name)))
         ((and (display-images-p) (image-type-available-p 'svg))
          (insert-image (create-image svg 'svg t))
          (insert "\n"))
         (t
          (insert svg "\n")))
        (goto-char (point-min))
        (special-mode))
      (display-buffer (current-buffer)))))

;;;###autoload
(defun jobwatch-respace ()
  "Show a re-spacing plan from `mint-jw respace'. Nothing is applied."
  (interactive)
  (let* ((got (jobwatch--call (list "respace" "--json" "--window" jobwatch-window)))
         (code (car got)))
    (unless (or (= code 0) (= code 1))
      (error "mint-jw respace failed: %s" (cdr got)))
    (let ((plan (jobwatch--parse (cdr got))))
      (with-current-buffer (get-buffer-create "*jobwatch respace*")
        (let ((inhibit-read-only t))
          (erase-buffer)
          (insert (jobwatch--plan-text plan))
          (goto-char (point-min))
          (special-mode))
        (display-buffer (current-buffer))))))

(define-key jobwatch-mode-map (kbd "RET") #'jobwatch-evidence)
(define-key jobwatch-mode-map (kbd "e") #'jobwatch-evidence)
(define-key jobwatch-mode-map (kbd "p") #'jobwatch-plot)
(define-key jobwatch-mode-map (kbd "s") #'jobwatch-respace)
(define-key jobwatch-mode-map (kbd "t") #'jobwatch-timeline)

(provide 'jobwatch)
;;; jobwatch.el ends here
