//+------------------------------------------------------------------+
//|                                         ExportTradingDeals.mq5   |
//|                             Copyright 2026, Alin Soare           |
//+------------------------------------------------------------------+
#property copyright "Alin Soare"
#property link      "https://www.mql5.com/en/market/product/177780"
#property version   "1.00"
#property icon      "ExportTradingDeals.ico"
#property description "Exports your full deal history to a CSV file readable by the trading-stats desktop application."
#property description ""
#property description "Output file: MQL5\\Files\\trading_stats\\deals_<login>.csv"
#property description "Re-run at any time to refresh — the file is always overwritten with the latest history."
#property description ""
#property description "Compatible with Trading Stats: https://github.com/alinsoare/trading-stats"
#property script_show_inputs

/// History range: from=0 selects from earliest available server time.
input datetime InpHistoryFrom = 0;
/// If true (default), export through current server time. If false, use InpHistoryToCustom.
input bool InpHistoryToNow = true;
/// End of range when InpHistoryToNow is false (ignored when InpHistoryToNow is true). 0 = TimeCurrent().
input datetime InpHistoryToCustom = 0;

//+------------------------------------------------------------------+
//| Script program start function                                    |
//+------------------------------------------------------------------+
void OnStart()
  {
   datetime t_to;
   if(InpHistoryToNow)
      t_to = TimeCurrent();
   else
      t_to = (InpHistoryToCustom == 0 ? TimeCurrent() : InpHistoryToCustom);
   datetime t_from = InpHistoryFrom;

   if(!HistorySelect(t_from, t_to))
     {
      Print("ExportTradingDeals: HistorySelect failed from=", t_from, " to=", t_to);
      return;
     }

   const string subdir = "trading_stats";
   if(!FolderCreate(subdir, 0))
     {
      // May already exist; ignore if subsequent FileOpen works.
     }

   long login = AccountInfoInteger(ACCOUNT_LOGIN);
   string server = AccountInfoString(ACCOUNT_SERVER);

   // One fixed file per account — always overwritten (FILE_WRITE truncates).
   string fname = StringFormat("%s\\deals_%I64d.csv", subdir, login);

   // FILE_ANSI: portable on builds where FILE_UTF8 is missing (Wine/older terminals).
   int h = FileOpen(fname, FILE_WRITE | FILE_CSV | FILE_ANSI, ',');
   if(h == INVALID_HANDLE)
     {
      Print("ExportTradingDeals: cannot open ", fname, " err=", GetLastError());
      return;
     }

   // Header: dot decimals via DoubleToString; encoding is system ANSI (dashboard uses lossy UTF-8 read).
   FileWrite(h,
             "ticket", "time", "time_msc", "type", "entry", "symbol", "volume", "price",
             "commission", "swap", "profit", "magic", "comment", "position_id", "reason",
             "login", "server", "is_non_trade");

   const int total = HistoryDealsTotal();
   for(int i = 0; i < total; i++)
     {
      ulong ticket = HistoryDealGetTicket(i);
      if(ticket == 0)
         continue;

      datetime dtime = (datetime)HistoryDealGetInteger(ticket, DEAL_TIME);
      long time_msc = HistoryDealGetInteger(ticket, DEAL_TIME_MSC);
      ENUM_DEAL_TYPE dtype = (ENUM_DEAL_TYPE)HistoryDealGetInteger(ticket, DEAL_TYPE);
      ENUM_DEAL_ENTRY dentry = (ENUM_DEAL_ENTRY)HistoryDealGetInteger(ticket, DEAL_ENTRY);
      string sym = HistoryDealGetString(ticket, DEAL_SYMBOL);
      double vol = HistoryDealGetDouble(ticket, DEAL_VOLUME);
      double price = HistoryDealGetDouble(ticket, DEAL_PRICE);
      double comm = HistoryDealGetDouble(ticket, DEAL_COMMISSION);
      double swap = HistoryDealGetDouble(ticket, DEAL_SWAP);
      double profit = HistoryDealGetDouble(ticket, DEAL_PROFIT);
      long magic = HistoryDealGetInteger(ticket, DEAL_MAGIC);
      string comment = HistoryDealGetString(ticket, DEAL_COMMENT);
      long pos_id = HistoryDealGetInteger(ticket, DEAL_POSITION_ID);
      ENUM_DEAL_REASON reason = (ENUM_DEAL_REASON)HistoryDealGetInteger(ticket, DEAL_REASON);

      int is_non_trade = IsNonTradeDeal(dtype) ? 1 : 0;

      string tstr = TimeToString(dtime, TIME_DATE | TIME_SECONDS);
      string cmt = comment;
      StringReplace(cmt, ",", ";");
      StringReplace(cmt, "\n", " ");
      StringReplace(cmt, "\r", " ");

      FileWrite(h,
                (string)ticket,
                tstr,
                (string)time_msc,
                (string)((int)dtype),
                (string)((int)dentry),
                sym,
                DoubleToString(vol, 8),
                DoubleToString(price, 8),
                DoubleToString(comm, 8),
                DoubleToString(swap, 8),
                DoubleToString(profit, 8),
                (string)magic,
                cmt,
                (string)pos_id,
                (string)((int)reason),
                (string)login,
                server,
                (string)is_non_trade);
     }

   FileClose(h);
   Print("ExportTradingDeals: wrote ", total, " rows to ", fname);
  }

//+------------------------------------------------------------------+
bool IsNonTradeDeal(const ENUM_DEAL_TYPE t)
  {
   return (t == DEAL_TYPE_BALANCE ||
           t == DEAL_TYPE_CREDIT ||
           t == DEAL_TYPE_CHARGE ||
           t == DEAL_TYPE_CORRECTION ||
           t == DEAL_TYPE_BONUS ||
           t == DEAL_TYPE_INTEREST ||
           t == DEAL_TYPE_BUY_CANCELED ||
           t == DEAL_TYPE_SELL_CANCELED);
  }
