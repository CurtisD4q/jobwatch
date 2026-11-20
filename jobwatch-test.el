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
