;;; jobwatch-test.el --- fixed examples for jobwatch -*- lexical-binding: t -*-

(require 'ert)
(require 'jobwatch)

(defconst jobwatch-test-report
  "{\"jobs\":[{\"name\":\"backup\",\"state\":\"failing\",\"level\":\"problem\",\"says\":\"Its last run failed.\",\"runs\":4,\"durations\":[600,660,900]}],\"findings\":[{\"job\":\"backup\",\"kind\":\"failure\",\"level\":\"problem\",\"says\":\"Its last run failed.\",\"evidence\":[\"When: Tuesday\",\"Stopped at exit 1\"]}]}")

(defconst jobwatch-test-plan
  "{\"moves\":[{\"job\":\"backup\",\"from\":\"02:00\",\"to\":\"04:00\"}],\"before\":\"40 min\",\"after\":\"0 min\",\"applied\":false,\"effect\":\"Health check before: three jobs at once. After the move: no three jobs run at once.\"}")

(ert-deftest jobwatch-table-from-report ()
  (let ((jobs (jobwatch--ingest jobwatch-test-report))
        (rows nil))
    (setq rows (jobwatch--rows jobs))
    (should (equal (car (car rows)) "backup"))
    (should (equal (aref (cdr (car rows)) 4) "Its last run failed."))))

(ert-deftest jobwatch-evidence-is-the-finding ()
  (jobwatch--ingest jobwatch-test-report)
  (let ((text (jobwatch--evidence-text "backup")))
    (should (string-match-p "Its last run failed\\." text))
    (should (string-match-p "When: Tuesday" text))
    (should (string-match-p "Stopped at exit 1" text))
    (should-not (string-match-p "mint-jw" text))))

(ert-deftest jobwatch-svg-plots-lengths ()
  (let ((svg (jobwatch--svg "backup" '(600 660 900))))
    (should (string-match-p "<svg" svg))
    (should (string-match-p "backup" svg))
    (should (string-match-p "polyline" svg))
    (should (null (jobwatch--svg "backup" nil)))))

(ert-deftest jobwatch-respace-plan-is-not-applied ()
  (let ((text (jobwatch--plan-text (jobwatch--parse jobwatch-test-plan))))
    (should (string-match-p "Nothing was changed\\." text))
    (should (string-match-p "backup  02:00 -> 04:00" text))
    (should (string-match-p "After the move" text))
    (should-not (string-match-p "--on" text))))

(provide 'jobwatch-test)
;;; jobwatch-test.el ends here

;; The day

(defconst jobwatch-test-timeline
  "{\"clock\":\"00          06          12          18        24\",\"bars\":[{\"name\":\"backup\",\"state\":\"fine\",\"period\":1440,\"start\":120,\"usual\":150,\"slow\":150,\"next\":240,\"nextName\":\"sync\",\"gap\":60,\"overrun\":false},{\"name\":\"sync\",\"state\":\"slowing\",\"says\":\"It is 31% slower per GB than six weeks ago.\",\"period\":1440,\"start\":240,\"usual\":270,\"slow\":300,\"next\":-1,\"gap\":0,\"overrun\":false}]}")

(defconst jobwatch-test-timeline-overrun
  "{\"clock\":\"00          06          12          18        24\",\"bars\":[{\"name\":\"backup\",\"state\":\"overrunning\",\"period\":1440,\"start\":120,\"usual\":150,\"slow\":250,\"next\":240,\"nextName\":\"sync\",\"gap\":0,\"overrun\":true}]}")

(ert-deftest jobwatch-timeline-shows-a-day ()
  (let* ((jobwatch--timeline (jobwatch--parse jobwatch-test-timeline))
         (text (jobwatch--timeline-text)))
    ;; Both jobs, with their clock times.
    (should (string-match-p "backup" text))
    (should (string-match-p "sync" text))
    (should (string-match-p "02:00" text))
    (should (string-match-p "02:30" text))
    (should (string-match-p "04:00" text))
    ;; The gap before the next job, said the way people say it.
    (should (string-match-p "1.0 hours before sync" text))
    ;; A job nothing follows says so rather than showing a blank.
    (should (string-match-p "no job starts within 12 hours" text))
    ;; The state word comes from the same report, so it is the same word.
    (should (string-match-p "slowing" text))
    ;; The axis is the one the command drew.
    (should (string-match-p "18" text))))

(ert-deftest jobwatch-timeline-says-when-a-job-runs-into-the-next ()
  (let* ((jobwatch--timeline (jobwatch--parse jobwatch-test-timeline-overrun))
         (text (jobwatch--timeline-text)))
    (should (string-match-p "runs into sync" text))
    (should-not (string-match-p "before sync" text))))

(ert-deftest jobwatch-timeline-is-empty-not-broken ()
  (let ((jobwatch--timeline (jobwatch--parse "{\"clock\":\"\",\"bars\":[]}")))
    (should (string-match-p "No job has enough runs to draw yet" (jobwatch--timeline-text))))
  ;; A report with no timeline at all is the same as an empty one.
  (let ((jobwatch--timeline nil))
    (should (string-match-p "No job has enough runs" (jobwatch--timeline-text)))))

(ert-deftest jobwatch-timeline-keeps-the-clock-honest ()
  (should (equal (jobwatch--minutes 0) "00:00"))
  (should (equal (jobwatch--minutes 150) "02:30"))
  (should (equal (jobwatch--minutes 1440) "00:00+1d"))
  (should (equal (jobwatch--minutes 1470) "00:30+1d"))
  ;; A run that starts at 23:00 and ends at 01:00 reads as the next day.
  (should (equal (jobwatch--minutes 1500) "01:00+1d")))

(ert-deftest jobwatch-timeline-marks-a-weekly-job ()
  (let* ((jobwatch--timeline
          (jobwatch--parse
           "{\"clock\":\"\",\"bars\":[{\"name\":\"week\",\"state\":\"fine\",\"period\":10080,\"start\":120,\"usual\":150,\"slow\":250,\"next\":240,\"nextName\":\"daily\",\"gap\":0,\"overrun\":true}]}"))
         (text (jobwatch--timeline-text)))
    (should (string-match-p "week" text))
    (should (string-match-p "(weekly)" text))))
