#ifndef FIRSTSERVER_H
#define FIRSTSERVER_H

#include <QObject>

class FirstServer : public QObject
{
    Q_OBJECT
public:
    explicit FirstServer(QObject *parent = 0);

signals:

public slots:
};

#endif // FIRSTSERVER_H