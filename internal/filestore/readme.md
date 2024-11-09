FileStore implements a thread-safe gateway to file
retrieval, and scheduling of parallel background downloads,
prioritizing those files that are awaited for by the caller
(tester).

Method Schedule(sha256, url) schedules a download of the
file with the given sha256 hash of decompressed file
contents and the given URL if the file does not yet exist at
the time of download attempt.

At the time of schedule, an incremental counter value is
assigned to the (key, url) pair. For each key we store all
the download URLs with their respective counter values and
also the largest assigned counter value separately. We also
store the progress of the download attempts for this key -
the highest ATTEMPTED URL counter value.

If we don't know the sha256 hash of the file contents, we
must use method ScheduledUnknown(url) to schedule a download
of the file. The method would return a uuid and schedule the
file download in the background. The uuid can be used to
track the progress of the download.

Method Await(file key) checks, whether the file exists and
returns it if it does. Otherwise it locks the state of the
file store, checks again, whether the file exists and
returns it if it does. If the file does not exist, it
captures the urls that are yet to be tried for downloading
the file as a scalar value assigned to the (key, url) pair
during scheduling, marks the file as awaited for and unlocks
the state of the file store. It then polls for events that
may signal file existence and compares the counter value of
progress of download attempts for this file or if the file
already exists it returns the file with no error. If the
counter reaches the largest assigned counter value at the
time of the await call, it means that all the provided URLs
have failed at downloading the file or an internal error has
occurred.

Method Await(file key) checks, whether the file exists and
returns it if it does. Otherwise it locks the state of file
store, checks how many files we have to

Multiple downloads can be performed in parallel, throttled
by a constant buffer size of channel. Each individual
download file must not be larger than 50MB when
decompressed. I think we can live with that limitation.

The file store is also responsible for file integrity
verification. The key of the file is actually a sha256 hash
of the decompressed file contents.

The file store also tracks disk usage and removes files when
total disk usage surpasses some treshold.

In the future, we could download the file, decompress it,
verify its sha256 hash and only if it matches, compress it
again and store it in the file store. This would actually
save disk space and increase access speed.

We could also in the future track the files stored by the
user. We can then limit the amount of space per user.