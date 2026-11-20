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

(provide 'jobwatch)
;;; jobwatch.el ends here
