package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

const installConcurrency = 4

func installPackages(ctx context.Context, packages []*Package, action func(int) error) error {
	goIndex := -1
	paceIndex := -1

	for index, pkg := range packages {
		switch pkg.Definition.Source {
		case "go":
			goIndex = index
		case "pace":
			paceIndex = index
		}
	}

	dependent := goIndex >= 0 && paceIndex >= 0
	jobs := make(chan int, len(packages))

	// Start Go first and run PACE in the same worker afterward, without tying up a waiting worker.
	if dependent {
		jobs <- goIndex
	}

	for index := range packages {
		if dependent && (index == goIndex || index == paceIndex) {
			continue
		}

		jobs <- index
	}

	close(jobs)

	failures := make([]error, len(packages))

	var workers sync.WaitGroup

	for range min(installConcurrency, len(jobs)) {
		workers.Go(func() {
			for index := range jobs {
				if ctx.Err() != nil {
					return
				}

				pkg := packages[index]

				err := action(index)
				if err != nil {
					failures[index] = fmt.Errorf("%s: %w", pkg.Name(), err)
				}

				if !dependent || index != goIndex {
					continue
				}

				pace := packages[paceIndex]

				if err != nil {
					failures[paceIndex] = fmt.Errorf("%s: skipped because go failed", pace.Name())

					continue
				}

				if ctx.Err() != nil {
					return
				}

				err = action(paceIndex)
				if err != nil {
					failures[paceIndex] = fmt.Errorf("%s: %w", pace.Name(), err)
				}
			}
		})
	}

	workers.Wait()

	return errors.Join(errors.Join(failures...), ctx.Err())
}
